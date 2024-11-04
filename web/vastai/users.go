package vastai

import (
	"context"
	"net/http"
)

func (c *Client) Whoami(ctx context.Context) (*User, error) {
	u, err := doJSON[User](ctx, c, http.MethodGet, "/v0/users/current/", http.StatusOK)
	return &u, err
}

type User struct {
	CanPay                  bool    `json:"can_pay"`
	ID                      int     `json:"id"`
	APIKey                  string  `json:"api_key"`
	Username                string  `json:"username"`
	SSHKey                  any     `json:"ssh_key"`
	PhoneNumber             any     `json:"phone_number"`
	PaypalEmail             any     `json:"paypal_email"`
	WiseEmail               any     `json:"wise_email"`
	Fullname                any     `json:"fullname"`
	BalanceThreshold        float64 `json:"balance_threshold"`
	BalanceThresholdEnabled bool    `json:"balance_threshold_enabled"`
	AutobillThreshold       any     `json:"autobill_threshold"`
	AutobillAmount          any     `json:"autobill_amount"`
	BilladdressLine1        any     `json:"billaddress_line1"`
	BilladdressLine2        any     `json:"billaddress_line2"`
	BilladdressCity         any     `json:"billaddress_city"`
	BilladdressZip          any     `json:"billaddress_zip"`
	BilladdressCountry      any     `json:"billaddress_country"`
	BillingCreditonly       int     `json:"billing_creditonly"`
	BilladdressTaxinfo      any     `json:"billaddress_taxinfo"`
	PasswordResettable      any     `json:"password_resettable"`
	Email                   string  `json:"email"`
	HasBilling              bool    `json:"has_billing"`
	HasPayout               bool    `json:"has_payout"`
	HostOnly                bool    `json:"host_only"`
	HostAgreementAccepted   bool    `json:"host_agreement_accepted"`
	EmailVerified           bool    `json:"email_verified"`
	Last4                   any     `json:"last4"`
	Balance                 int     `json:"balance"`
	Credit                  float64 `json:"credit"`
	GotSignupCredit         int     `json:"got_signup_credit"`
	User                    string  `json:"user"`
	PaidVerified            float64 `json:"paid_verified"`
	PaidExpected            float64 `json:"paid_expected"`
	BilledVerified          float64 `json:"billed_verified"`
	BilledExpected          float64 `json:"billed_expected"`
	Rights                  any     `json:"rights"`
	Sid                     string  `json:"sid"`
}
