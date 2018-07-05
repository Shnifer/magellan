package main

//COPYPASTE: in Pilot

import (
	. "github.com/Shnifer/magellan/commons"
	. "github.com/Shnifer/magellan/draw"
	"github.com/Shnifer/magellan/graph"
	"github.com/Shnifer/magellan/v2"
	"golang.org/x/image/colornames"
)

type predictors struct {
	show            bool
	predictorZero   *TrackPredictor
	predictorThrust *TrackPredictor
}

func (p *predictors) init(cam *graph.Camera) {
	predictorSprite := NewAtlasSprite(PredictorAN, cam.Deny())
	predictorSprite.SetSize(20, 20)
	opts := TrackPredictorOpts{
		Cam:      cam,
		Sprite:   predictorSprite,
		Galaxy:   Data.Galaxy,
		Clr:      colornames.Palevioletred,
		Layer:    graph.Z_ABOVE_OBJECT + 1,
		UpdT:     0.2,
		NumInSec: 10,
		GravEach: 2,
		TrackLen: 60,
	}

	p.predictorThrust = NewTrackPredictor(opts)

	predictor2Sprite := NewAtlasSprite(PredictorAN, cam.Deny())
	predictor2Sprite.SetSize(15, 15)
	predictor2Sprite.SetColor(colornames.Darkgray)
	opts.Sprite = predictor2Sprite
	opts.Clr = colornames.Cadetblue
	p.predictorZero = NewTrackPredictor(opts)

	p.show = true
}

func (p predictors) setParams() {
	p.predictorThrust.SetAccelSessionTimeShipPos(Data.PilotData.ThrustVector, Data.PilotData.SessionTime, Data.PilotData.Ship)
	p.predictorZero.SetAccelSessionTimeShipPos(v2.ZV, Data.PilotData.SessionTime, Data.PilotData.Ship)
}

func (p predictors) Req() *graph.DrawQueue {
	R := graph.NewDrawQueue()
	if p.show {
		R.Append(p.predictorZero)
		R.Append(p.predictorThrust)
	}
	return R
}