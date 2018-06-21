package main

import (
	"bytes"
	"encoding/json"
	"github.com/Shnifer/magellan/commons"
	"github.com/Shnifer/magellan/v2"
	"image/color"
	"io/ioutil"
	"strconv"
	"strings"
)

type fileData struct {
	ID, Parent  string
	Diameter    float64
	Distance    float64
	MaxGravity  float64
	GravityR10  float64
	OrbitPeriod float64
	Color       struct{ R, G, B byte }
	Count       int
	//начальный угол, если объект 1
	StartAng float64
	//отклонения от базовых значений в процентах, если объектов много
	RadMassDev     float64
	PeriodOrbitDev float64
	Emissions      []commons.Emission
	TexName        string
}

const DEFType = "planet"
const K_OrbitPeriod = 300.0
const K_Radius = 3.7
const K_Mass = 1.0
const A_Mass = 2.0
const K_ZDepth = 1.0
const K_Size = 0.0033 *10

func main() {
	buf, err := ioutil.ReadFile("galaxyPredata.json")
	if err != nil {
		panic(err)
	}

	var inData []fileData
	err = json.Unmarshal(buf, &inData)
	if err != nil {
		panic(err)
	}

	outData := commons.Galaxy{
		Points: make(map[string]*commons.GalaxyPoint),
	}
	maxOrbit := 0.0

	for _, v := range inData {
		if v.Count == 1 {
			if v.Distance > maxOrbit {
				maxOrbit = v.Distance
			}
			gp, id := createGP(v)
			outData.Points[id] = gp
		} else {
			for i := 0; i < v.Count; i++ {
				w := v
				w.ID = v.ID + "-" + strconv.Itoa(i)

				kPeriodOrbit := commons.KDev(v.PeriodOrbitDev)
				w.OrbitPeriod *= kPeriodOrbit
				w.Distance *= kPeriodOrbit

				kRadMass := commons.KDev(v.RadMassDev)
				w.Diameter *= kRadMass
				w.MaxGravity *= kRadMass
				w.GravityR10 *= kRadMass

				gp, id := createGP(w)
				outData.Points[id] = gp
			}
		}
	}

	outData.SpawnDistance = maxOrbit * 1.1 *K_Radius

	buf, err = json.Marshal(outData)
	if err != nil {
		panic(err)
	}
	buf = bytes.Replace(buf, []byte(`"Pos":{},`), []byte(""), -1)
	buf = bytes.Replace(buf, []byte("}},"), []byte("}},\n"), -1)
	//var idbuf bytes.Buffer
	//json.Indent(&idbuf, buf, "", " ")
	ioutil.WriteFile("galaxy_solar.json", buf, 0)
}

func createGP(v fileData) (*commons.GalaxyPoint, string) {
	objType := DEFType
	if v.TexName != "" {
		s := strings.Split(v.TexName, ".")
		objType = s[0]
	}

	clr := color.RGBA{R: v.Color.R, G: v.Color.G, B: v.Color.B, A: 255}

	okr := func(x float64) float64 {
		const sgn = 100
		return float64(int(x*sgn)) / sgn
	}

	zd:=v.GravityR10/3 * K_ZDepth
	maxGrav:=1-(1-v.MaxGravity)/A_Mass
	mass:=maxGrav*zd*zd

	gp := commons.GalaxyPoint{
		ParentID:  v.Parent,
		Pos:       v2.ZV,
		Orbit:     okr(v.Distance * K_Radius),
		Period:    okr(v.OrbitPeriod * K_OrbitPeriod),
		Type:      objType,
		Size:      okr(v.Diameter / 2 * K_Size),
		Color:     clr,
		Mass:      okr(mass * K_Mass),
		GDepth:    okr(zd),
		ScanData:  v.ID,
		Emissions: v.Emissions,
	}
	return &gp, v.ID
}
