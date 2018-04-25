package commons

import (
	"math/rand"
)

func Clamp(x, min, max float64) float64 {
	switch {
	case x > max:
		return max
	case x < min:
		return min
	default:
		return x
	}
}

var gravityConst float64
var warpGravityConst float64

func SetGravityConsts(G, W float64) {
	gravityConst = G
	warpGravityConst = W
}

//
func Gravity(mass, lenSqr, zDist float64) float64 {
	d2 := lenSqr + zDist*zDist
	//d2 = d2 * d2
	if d2 == 0 {
		return 0
	}
	return gravityConst * mass / d2
}

func WarpGravity(mass, lenSqr, velSqr, zDist float64) float64 {

	d2 := lenSqr + zDist*zDist
	d2 = d2 * d2
	if d2 == 0 {
		return 0
	}

	return warpGravityConst * mass * lenSqr / d2 * (1 + velSqr)
}

//Возвращает коэффициент нормальной дистрибуций
//сигма в процентах devProcent
//68% попадут в (100-devProcent, 100+devProcent)
//95% попадут в (100-2*devProcent, 100+2*devProcent)
//Отклонения больше 3 сигма ограничиваются
func KDev(devProcent float64) float64 {
	r := rand.NormFloat64()
	if r > 3 {
		r = 3
	}
	if r < (-3) {
		r = -3
	}
	r = 1 + r*devProcent/100
	if r < 0 {
		r = 0.00001
	}
	return r
}
