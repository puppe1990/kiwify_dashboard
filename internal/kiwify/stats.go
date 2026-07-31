package kiwify

import "context"

// StatsQuery filters GET /stats.
type StatsQuery struct {
	StartDate string
	EndDate   string
	ProductID string
}

func (q StatsQuery) queryMap() map[string]string {
	m := map[string]string{}
	if q.StartDate != "" {
		m["start_date"] = q.StartDate
	}
	if q.EndDate != "" {
		m["end_date"] = q.EndDate
	}
	if q.ProductID != "" {
		m["product_id"] = q.ProductID
	}
	return m
}

// Stats is the GET /stats response (sales statistics).
// total_net_amount is in minor units (centavos). Rates are percentages.
type Stats struct {
	CreditCardApprovalRate float64 `json:"credit_card_approval_rate"`
	TotalSales             float64 `json:"total_sales"`
	TotalNetAmount         float64 `json:"total_net_amount"`
	RefundRate             float64 `json:"refund_rate"`
	ChargebackRate         float64 `json:"chargeback_rate"`
	TotalBoletoGenerated   float64 `json:"total_boleto_generated"`
	TotalBoletoPaid        float64 `json:"total_boleto_paid"`
	BoletoRate             float64 `json:"boleto_rate"`
}

// SalesStats fetches sales statistics for the given date range (and optional product).
func (c *Client) SalesStats(ctx context.Context, q StatsQuery) (Stats, error) {
	var out Stats
	if err := c.GetJSON(ctx, "/stats", q.queryMap(), &out); err != nil {
		return Stats{}, err
	}
	return out, nil
}
