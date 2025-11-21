package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/models"
)

var (
	ErrDuplicateReward = errors.New("reward event already exists")
)

type RewardCreationParams struct {
	UserID     uuid.UUID
	Symbol     string
	Shares     decimal.Decimal
	GrantPrice decimal.Decimal
	Brokerage  decimal.Decimal
	Taxes      decimal.Decimal
	Total      decimal.Decimal
	EventKey   string
	RewardedAt time.Time
}

func (r *Repository) CreateReward(ctx context.Context, params RewardCreationParams) (*models.RewardEvent, error) {
	var result models.RewardEvent
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.Serializable}, func(tx pgx.Tx) error {
		if err := r.ensureUser(ctx, tx, params.UserID); err != nil {
			return err
		}
		if err := r.ensureStock(ctx, tx, params.Symbol); err != nil {
			return err
		}

		reward, err := r.insertReward(ctx, tx, params)
		if err != nil {
			return err
		}

		if err := r.insertLedgerEntries(ctx, tx, reward, params); err != nil {
			return err
		}

		if err := r.updateUserPosition(ctx, tx, reward, params); err != nil {
			return err
		}

		result = *reward
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateReward) {
			return nil, err
		}
		return nil, err
	}
	return &result, nil
}

func (r *Repository) ensureUser(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO users (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`, userID)
	return err
}

func (r *Repository) ensureStock(ctx context.Context, tx pgx.Tx, symbol string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO stocks (symbol, name, exchange, status)
		VALUES ($1, $2, 'NSE', 'ACTIVE')
		ON CONFLICT (symbol) DO NOTHING
	`, strings.ToUpper(symbol), strings.ToUpper(symbol))
	return err
}

func (r *Repository) insertReward(ctx context.Context, tx pgx.Tx, params RewardCreationParams) (*models.RewardEvent, error) {
	id := uuid.New()
	row := tx.QueryRow(ctx, `
		INSERT INTO reward_events (
			id, user_id, symbol, shares, granted_price_inr, brokerage_inr,
			taxes_inr, total_cash_out_inr, rewarded_at, event_key
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, user_id, symbol, shares, granted_price_inr, brokerage_inr,
		          taxes_inr, total_cash_out_inr, rewarded_at, created_at, event_key
	`, id, params.UserID, strings.ToUpper(params.Symbol), decimalToNumeric(params.Shares),
		decimalToNumeric(params.GrantPrice), decimalToNumeric(params.Brokerage),
		decimalToNumeric(params.Taxes), decimalToNumeric(params.Total), params.RewardedAt, params.EventKey)

	var (
		reward models.RewardEvent
		shares pgtype.Numeric
		price  pgtype.Numeric
		brk    pgtype.Numeric
		tax    pgtype.Numeric
		total  pgtype.Numeric
	)

	err := row.Scan(
		&reward.ID,
		&reward.UserID,
		&reward.Symbol,
		&shares,
		&price,
		&brk,
		&tax,
		&total,
		&reward.RewardedAt,
		&reward.CreatedAt,
		&reward.EventKey,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateReward
		}
		return nil, err
	}

	reward.Shares = numericToDecimal(shares)
	reward.GrantedPrice = numericToDecimal(price)
	reward.BrokerageInr = numericToDecimal(brk)
	reward.TaxesInr = numericToDecimal(tax)
	reward.TotalCashOut = numericToDecimal(total)
	return &reward, nil
}

func (r *Repository) insertLedgerEntries(ctx context.Context, tx pgx.Tx, reward *models.RewardEvent, params RewardCreationParams) error {
	stockAccount := fmt.Sprintf("stock_inventory:%s", reward.Symbol)
	if err := r.insertLedgerEntry(ctx, tx, LedgerEntryParams{
		EventID:     reward.ID,
		AccountCode: stockAccount,
		AccountType: "asset",
		Symbol:      reward.Symbol,
		Debit:       params.GrantPrice.Mul(params.Shares),
		Credit:      decimal.Zero,
		StockUnits:  params.Shares,
		Memo:        "Rewarded stock inventory",
	}); err != nil {
		return err
	}

	if err := r.insertLedgerEntry(ctx, tx, LedgerEntryParams{
		EventID:     reward.ID,
		AccountCode: "brokerage_expense",
		AccountType: "expense",
		Debit:       params.Brokerage,
		Memo:        "Brokerage charges",
	}); err != nil {
		return err
	}

	if err := r.insertLedgerEntry(ctx, tx, LedgerEntryParams{
		EventID:     reward.ID,
		AccountCode: "tax_expense",
		AccountType: "expense",
		Debit:       params.Taxes,
		Memo:        "Statutory taxes",
	}); err != nil {
		return err
	}

	totalCash := params.Total
	if err := r.insertLedgerEntry(ctx, tx, LedgerEntryParams{
		EventID:     reward.ID,
		AccountCode: "cash",
		AccountType: "asset",
		Credit:      totalCash,
		Memo:        "Cash outflow for reward",
	}); err != nil {
		return err
	}
	return nil
}

type LedgerEntryParams struct {
	EventID     uuid.UUID
	AccountCode string
	AccountType string
	Symbol      string
	Debit       decimal.Decimal
	Credit      decimal.Decimal
	StockUnits  decimal.Decimal
	Memo        string
}

func (r *Repository) insertLedgerEntry(ctx context.Context, tx pgx.Tx, params LedgerEntryParams) error {
	accountID, err := r.ensureLedgerAccount(ctx, tx, params.AccountCode, params.AccountType, params.Symbol)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (event_id, account_id, debit_inr, credit_inr, stock_units, memo)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, params.EventID, accountID,
		decimalToNumeric(params.Debit),
		decimalToNumeric(params.Credit),
		decimalToNumeric(params.StockUnits),
		params.Memo)
	return err
}

func (r *Repository) ensureLedgerAccount(ctx context.Context, tx pgx.Tx, code, accountType, symbol string) (int, error) {
	var accountID int
	err := tx.QueryRow(ctx, `
		INSERT INTO ledger_accounts (code, type, symbol)
		VALUES ($1,$2,$3)
		ON CONFLICT (code)
		DO UPDATE SET symbol = COALESCE(ledger_accounts.symbol, EXCLUDED.symbol)
		RETURNING id
	`, code, accountType, nullableString(symbol)).Scan(&accountID)
	return accountID, err
}

func (r *Repository) updateUserPosition(ctx context.Context, tx pgx.Tx, reward *models.RewardEvent, params RewardCreationParams) error {
	var (
		currentShares pgtype.Numeric
		currentAvg    pgtype.Numeric
	)
	err := tx.QueryRow(ctx, `
		SELECT net_shares, avg_cost_inr
		FROM user_positions
		WHERE user_id = $1 AND symbol = $2
		FOR UPDATE
	`, reward.UserID, reward.Symbol).Scan(&currentShares, &currentAvg)

	costBasis := params.GrantPrice.Mul(params.Shares)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, err = tx.Exec(ctx, `
				INSERT INTO user_positions (user_id, symbol, net_shares, avg_cost_inr)
				VALUES ($1,$2,$3,$4)
			`, reward.UserID, reward.Symbol, decimalToNumeric(params.Shares), decimalToNumeric(params.GrantPrice))
			return err
		}
		return err
	}

	currentSharesDec := numericToDecimal(currentShares)
	currentAvgDec := numericToDecimal(currentAvg)
	newShares := currentSharesDec.Add(params.Shares)

	var newAvg decimal.Decimal
	if newShares.GreaterThan(decimal.Zero) {
		totalCost := currentAvgDec.Mul(currentSharesDec).Add(costBasis)
		newAvg = totalCost.Div(newShares)
	} else {
		newAvg = decimal.Zero
	}

	_, err = tx.Exec(ctx, `
		UPDATE user_positions
		SET net_shares = $3, avg_cost_inr = $4, updated_at = NOW()
		WHERE user_id = $1 AND symbol = $2
	`, reward.UserID, reward.Symbol, decimalToNumeric(newShares), decimalToNumeric(newAvg))
	return err
}

func nullableString(val string) any {
	if val == "" {
		return nil
	}
	return val
}
