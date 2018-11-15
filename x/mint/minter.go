package mint

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// current inflation state
type Minter struct {
	LastInflation       time.Time `json:"last_inflation"`        // time of the last inflation
	LastInflationChange time.Time `json:"last_inflation_change"` // time which the last inflation rate change
	Inflation           sdk.Dec   `json:"inflation"`             // current annual inflation rate
	HourlyProvisions    sdk.Int   `json:"hourly_provisions"`     // current hourly provisions rate
}

// minter object for a new minter
func InitialMinter() Minter {
	return Minter{
		LastInflation:       time.Unix(0, 0),
		LastInflationChange: time.Unix(0, 0),
		Inflation:           sdk.NewDecWithPrec(13, 2),
		HourlyProvisions:    sdk.NewInt(0),
	}
}

func validateMinter(minter Minter) error {
	if minter.Inflation.LT(sdk.ZeroDec()) {
		return fmt.Errorf("mint parameter Inflation should be positive, is %s ", minter.Inflation.String())
	}
	if minter.Inflation.GT(sdk.OneDec()) {
		return fmt.Errorf("mint parameter Inflation must be <= 1, is %s", minter.Inflation.String())
	}
	return nil
}

var hrsPerYr = sdk.NewDec(8766) // as defined by a julian year of 365.25 days

// process provisions for an hour period
// NOTE if ProcessProvisions is called for the first time
//      from an InitialMinter, ProcessProvisions will
//      effectively only set the blocktime as the default
//      HourlyProvisions is 0.
func (m Minter) ProcessProvisions(params Params, blockTime time.Time) (
	minter Minter, provisions sdk.Coin) {

	dur := m.LastInflation.Sub(blockTime).Nanoseconds()
	portionOfHour := dur / time.Hour.Nanoseconds()

	provisionsAmt := m.HourlyProvisions.MulRaw(portionOfHour)
	provisions = sdk.NewCoin(params.MintDenom, provisionsAmt)

	minter.LastInflation = blockTime

	return m, provisions
}

// get the new inflation rate for the next hour
func (m Minter) NextInflation(params Params, bondedRatio sdk.Dec) (inflation sdk.Dec) {

	// The target annual inflation rate is recalculated for each previsions cycle. The
	// inflation is also subject to a rate change (positive or negative) depending on
	// the distance from the desired ratio (67%). The maximum rate change possible is
	// defined to be 13% per year, however the annual inflation is capped as between
	// 7% and 20%.

	// (1 - bondedRatio/GoalBonded) * InflationRateChange
	inflationRateChangePerYear := sdk.OneDec().
		Sub(bondedRatio.Quo(params.GoalBonded)).
		Mul(params.InflationRateChange)
	inflationRateChange := inflationRateChangePerYear.Quo(hrsPerYr)

	// increase the new annual inflation for this next cycle
	inflation = m.Inflation.Add(inflationRateChange)
	if inflation.GT(params.InflationMax) {
		inflation = params.InflationMax
	}
	if inflation.LT(params.InflationMin) {
		inflation = params.InflationMin
	}

	return inflation
}

// get the new hourly inflation provisions rate
func (m Minter) NextHourlyProvisions(params Params, totalSupply sdk.Dec) (provisions sdk.Int) {
	provisionsDec := m.Inflation.Mul(totalSupply).Quo(hrsPerYr)
	return provisionsDec.TruncateInt()
}
