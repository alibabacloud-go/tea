package tea

import (
	"math"
	"math/rand"
)

type BackoffPolicy interface {
	GetDelayTimeMillis(ctx *RetryPolicyContext) *int64
}

func NewBackoffPolicy(options map[string]interface{}) (backoffPolicy BackoffPolicy) {
	policy := StringValue(TransInterfaceToString(options["policy"]))
	switch policy {
	case "Fixed":
		backoffPolicy = &FixedBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
		}
		return
	case "Random":
		backoffPolicy = &RandomBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
			Cap:    TransInterfaceToInt64(options["cap"]),
		}
		return
	case "Exponential":
		backoffPolicy = &ExponentialBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
			Cap:    TransInterfaceToInt64(options["cap"]),
		}
		return
	case "EqualJitter":
		backoffPolicy = &EqualJitterBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
			Cap:    TransInterfaceToInt64(options["cap"]),
		}
		return
	case "ExponentialWithEqualJitter":
		backoffPolicy = &EqualJitterBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
			Cap:    TransInterfaceToInt64(options["cap"]),
		}
		return
	case "FullJitter":
		backoffPolicy = &FullJitterBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
			Cap:    TransInterfaceToInt64(options["cap"]),
		}
		return
	case "ExponentialWithFullJitter":
		backoffPolicy = &FullJitterBackoffPolicy{
			Period: TransInterfaceToInt(options["period"]),
			Cap:    TransInterfaceToInt64(options["cap"]),
		}
		return
	}
	return nil
}

type FixedBackoffPolicy struct {
	Period *int
}

func (fixedBackoff *FixedBackoffPolicy) GetDelayTimeMillis(ctx *RetryPolicyContext) *int64 {
	return Int64(int64(IntValue(fixedBackoff.Period)))
}

type RandomBackoffPolicy struct {
	Period *int
	Cap    *int64
}

func (randomBackoff *RandomBackoffPolicy) GetDelayTimeMillis(ctx *RetryPolicyContext) *int64 {
	if randomBackoff.Cap == nil {
		randomBackoff.Cap = Int64(20 * 1000)
	}
	ceil := math.Min(float64(*randomBackoff.Cap), float64(IntValue(randomBackoff.Period))*float64(IntValue(ctx.RetriesAttempted)))
	return Int64(int64(rand.Float64() * ceil))
}

type ExponentialBackoffPolicy struct {
	Period *int
	Cap    *int64
}

func (exponentialBackoff *ExponentialBackoffPolicy) GetDelayTimeMillis(ctx *RetryPolicyContext) *int64 {
	if exponentialBackoff.Cap == nil {
		exponentialBackoff.Cap = Int64(3 * 24 * 60 * 60 * 1000)
	}
	return Int64(int64(math.Min(float64(*exponentialBackoff.Cap), float64(IntValue(exponentialBackoff.Period))*math.Pow(2.0, float64(IntValue(ctx.RetriesAttempted))))))
}

type EqualJitterBackoffPolicy struct {
	Period *int
	Cap    *int64
}

func (equalJitterBackoff *EqualJitterBackoffPolicy) GetDelayTimeMillis(ctx *RetryPolicyContext) *int64 {
	if equalJitterBackoff.Cap == nil {
		equalJitterBackoff.Cap = Int64(3 * 24 * 60 * 60 * 1000)
	}
	ceil := math.Min(float64(*equalJitterBackoff.Cap), float64(IntValue(equalJitterBackoff.Period))*math.Pow(2.0, float64(IntValue(ctx.RetriesAttempted))))
	return Int64(int64(ceil/2 + rand.Float64()*(ceil/2+1)))
}

type FullJitterBackoffPolicy struct {
	Period *int
	Cap    *int64
}

func (fullJitterBackof *FullJitterBackoffPolicy) GetDelayTimeMillis(ctx *RetryPolicyContext) *int64 {
	if fullJitterBackof.Cap == nil {
		fullJitterBackof.Cap = Int64(3 * 24 * 60 * 60 * 1000)
	}
	ceil := math.Min(float64(*fullJitterBackof.Cap), float64(IntValue(fullJitterBackof.Period))*math.Pow(2.0, float64(IntValue(ctx.RetriesAttempted))))
	return Int64(int64(rand.Float64() * ceil))
}
