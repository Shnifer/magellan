package main

import (
	"github.com/Shnifer/magellan/commons"
	. "github.com/Shnifer/magellan/draw"
	"github.com/Shnifer/magellan/graph"
	. "github.com/Shnifer/magellan/log"
	"github.com/Shnifer/magellan/v2"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
	"golang.org/x/image/colornames"
)

const (
	shipSize = 1
)

type cosmoScene struct {
	ship     *graph.Sprite
	shipMark *graph.Sprite
	shipRB   *commons.RBFollower

	sessionTime  *commons.SessionTime
	lastPilotMsg int

	lastServerID int
	otherShips   map[string]*OtherShip

	caption *graph.Text
	cam     *graph.Camera

	objects map[string]*CosmoPoint

	scanner *scanner

	showPredictor   bool
	predictorZero   *TrackPredictor
	predictorThrust *TrackPredictor

	naviMarkerT float64
}

func newCosmoScene() *cosmoScene {
	caption := graph.NewText("Navi scene", Fonts[Face_cap], colornames.Aliceblue)
	caption.SetPosPivot(graph.ScrP(0.1, 0.1), graph.TopLeft())

	cam := graph.NewCamera()
	cam.Center = graph.ScrP(0.5, 0.5)
	cam.Recalc()

	ship := NewAtlasSprite(commons.ShipAN, cam.Phys())
	ship.SetSize(shipSize, shipSize)

	shipMark := NewAtlasSprite(commons.MARKShipAN, cam.FixS())

	predictorSprite := NewAtlasSprite(commons.PredictorAN, cam.Deny())
	predictorSprite.SetSize(20, 20)
	predictorThrust := NewTrackPredictor(cam, predictorSprite, &Data, Track_CurrentThrust, colornames.Palevioletred, graph.Z_ABOVE_OBJECT+1)

	predictor2Sprite := NewAtlasSprite(commons.PredictorAN, cam.Deny())
	predictor2Sprite.SetSize(15, 15)
	predictor2Sprite.SetColor(colornames.Darkgray)

	predictorZero := NewTrackPredictor(cam, predictor2Sprite, &Data, Track_ZeroThrust, colornames.Cadetblue, graph.Z_ABOVE_OBJECT)

	return &cosmoScene{
		caption:         caption,
		ship:            ship,
		shipMark:        shipMark,
		cam:             cam,
		objects:         make(map[string]*CosmoPoint),
		predictorThrust: predictorThrust,
		predictorZero:   predictorZero,
		otherShips:      make(map[string]*OtherShip),
	}
}

func (s *cosmoScene) Init() {
	defer LogFunc("cosmoScene.Init")()

	stateData := Data.GetStateData()

	s.objects = make(map[string]*CosmoPoint)
	s.otherShips = make(map[string]*OtherShip)
	s.naviMarkerT = 0
	s.lastServerID = 0
	s.scanner = newScanner(s.cam)
	s.shipRB = commons.NewRBFollower(float64(DEFVAL.PingPeriod) / 1000)
	s.sessionTime = commons.NewSessionTime(Data.PilotData.SessionTime)

	for id, pd := range stateData.Galaxy.Points {
		cosmoPoint := NewCosmoPoint(pd, s.cam.Phys())
		s.objects[id] = cosmoPoint
	}
}

func (s *cosmoScene) Update(dt float64) {
	defer LogFunc("cosmoScene.Update")()
	//PilotData Rigid Body emulation
	if Data.PilotData.MsgID != s.lastPilotMsg {
		s.shipRB.MoveTo(Data.PilotData.Ship)
		s.sessionTime.MoveTo(Data.PilotData.SessionTime)
		s.lastPilotMsg = Data.PilotData.MsgID
	}
	s.sessionTime.Update(dt)
	Data.Galaxy.Update(s.sessionTime.Get())

	if Data.ServerData.MsgID != s.lastServerID {
		s.actualizeOtherShips()
	}

	s.shipRB.Update(dt)
	ship := s.shipRB.RB()

	s.cam.Pos = ship.Pos
	s.cam.Recalc()

	//update actual otherShips
	for id := range s.otherShips {
		s.otherShips[id].Update(dt)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mousex, mousey := ebiten.CursorPosition()
		s.procMouseClick(v2.V2{X: float64(mousex), Y: float64(mousey)})
	}
	s.naviMarkerT -= dt
	if s.naviMarkerT < 0 {
		s.naviMarkerT = 0
		Data.NaviData.ActiveMarker = false
	}

	for id, co := range s.objects {
		if gp, ok := Data.Galaxy.Points[id]; ok {
			s.objects[id].Pos = gp.Pos
		}
		co.Update(dt)
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		s.cam.Scale *= 1 + dt
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		s.cam.Scale /= 1 + dt
	}

	s.ship.SetPosAng(ship.Pos, ship.Ang)
	s.shipMark.SetPosAng(ship.Pos, ship.Ang)

	s.scanner.update(ship.Pos, dt)
}

func (s *cosmoScene) Draw(image *ebiten.Image) {
	defer LogFunc("cosmoScene.Draw")()

	Q := graph.NewDrawQueue()

	Q.Append(s.scanner)

	for _, co := range s.objects {
		Q.Append(co)
	}

	for _, os := range s.otherShips {
		Q.Append(os)
	}

	if s.showPredictor {
		Q.Append(s.predictorThrust)
		Q.Append(s.predictorZero)
	}

	alphaMark, alphaSprite := MarkAlpha(shipSize/2.0, s.cam)
	if alphaMark > 0 && s.shipMark != nil {
		s.shipMark.SetAlpha(alphaMark)
		Q.Add(s.shipMark, graph.Z_HUD)
	}
	if alphaSprite > 0 && s.ship != nil {
		s.ship.SetAlpha(alphaSprite)
		Q.Add(s.ship, graph.Z_HUD)
	}

	Q.Run(image)
}

func (s *cosmoScene) actualizeOtherShips() {
	s.lastServerID = Data.ServerData.MsgID

	//Create new otherShip and move all to new positions
	for _, otherData := range Data.ServerData.OtherShips {
		otherShip, ok := s.otherShips[otherData.Id]
		if !ok {
			otherShip = NewOtherShip(s.cam.FixS(), otherData.Name, float64(DEFVAL.OtherShipElastic)/1000)
			s.otherShips[otherData.Id] = otherShip
		}
		otherShip.SetRB(otherData.Ship)
	}

	//check for lost otherShips to delete
	for id := range s.otherShips {
		found := false
		for _, otherData := range Data.ServerData.OtherShips {
			if otherData.Id == id {
				found = true
				break
			}
		}
		if !found {
			delete(s.otherShips, id)
		}
	}
}

func (s *cosmoScene) procMouseClick(scrPos v2.V2) {
	worldPos := s.cam.UnApply(scrPos)
	for id, obj := range Data.Galaxy.Points {
		if worldPos.Sub(obj.Pos).LenSqr() < (obj.Size * obj.Size) {
			s.scanner.clicked(s.objects[id])
			return
		}
	}
	Data.NaviData.ActiveMarker = true
	Data.NaviData.MarkerPos = worldPos
	s.naviMarkerT = DEFVAL.NaviMarketDuration
}

func (*cosmoScene) Destroy() {
}
