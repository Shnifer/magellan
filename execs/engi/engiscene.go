package main

import (
	. "github.com/Shnifer/magellan/commons"
	. "github.com/Shnifer/magellan/draw"
	"github.com/Shnifer/magellan/graph"
	. "github.com/Shnifer/magellan/log"
	"github.com/Shnifer/magellan/ranma"
	"github.com/Shnifer/magellan/v2"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
	"golang.org/x/image/colornames"
	"log"
	"time"
)

const (
	focus_main = iota
	focus_enterBoost
)

type BoostParams struct {
	NodeType   string  `json:"node_type"`
	BaseTime   float64 `json:"base_time"`
	AZBonus    float64 `json:"az_bonus"`
	AZDmg      float64 `json:"az_damage"`
	BoostPower float64 `json:"boost_percent"`
	Code       string  `json:"code"`
	Password   string  `json:"password"`
}

type engiScene struct {
	shipID string

	ranma      *ranma.Ranma
	background *graph.Sprite

	systemsMonitor *systemsMonitor

	q *graph.DrawQueue

	tick <-chan time.Time

	wormOut string

	local localCounters

	focus      int
	boostInput *TextInput

	dieTimeout float64

	boostList   map[string]BoostParams
	wrongBoostT float64
}

func newEngiScene() *engiScene {
	back := NewAtlasSpriteHUD(EngiBackgroundAN)
	back.SetSize(float64(WinW), float64(WinH))
	back.SetPivot(graph.TopLeft())

	r := ranma.NewRanma(DEFVAL.RanmaAddr, DEFVAL.DropOnRepair, DEFVAL.RanmaTimeoutMs, DEFVAL.RanmaHistoryDepth)

	res := engiScene{
		ranma:          r,
		background:     back,
		systemsMonitor: newSystemsMonitor(),
		q:              graph.NewDrawQueue(),
		tick:           time.Tick(time.Second),
		boostList:      make(map[string]BoostParams),
	}

	textPanel := NewAtlasSprite(TextPanelAN, graph.NoCam)
	textPanel.SetPos(graph.ScrP(0.5, 0))
	textPanel.SetPivot(graph.TopMiddle())
	size := graph.ScrP(0.6, 0.1)
	textPanel.SetSize(size.X, size.Y)
	res.boostInput = NewTextInput(textPanel, Fonts[Face_cap], colornames.White, graph.Z_HUD+1, res.onBoostTextInput)

	return &res
}

func (s *engiScene) Init() {
	defer LogFunc("engiScene.Init")()

	if s.shipID == Data.ShipID {
		return
	}
	s.shipID = Data.ShipID

	s.local = initLocal()
	initMedi(Data.ShipID)
	s.focus = focus_main

	for sysN := 0; sysN < SysCount; sysN++ {
		if s.ranma.GetIn(sysN) != Data.EngiData.InV[sysN] {
			s.ranma.SetIn(sysN, Data.EngiData.InV[sysN])
		}
	}

	s.wormOut = ""
	s.boostList = loadHyBoostList()
}

func (s *engiScene) Update(dt float64) {
	defer LogFunc("engiScene.Update")()

	s.dieTimeout -= dt
	if s.dieTimeout < 0 {
		s.dieTimeout = 0
	}
	s.wrongBoostT -= dt
	if s.wrongBoostT < 0 {
		s.wrongBoostT = 0
	}

	Data.Galaxy.Update(Data.PilotData.SessionTime)
	updateBoosts(dt)

	if s.focus == focus_enterBoost {
		s.boostInput.Update(dt)
	}

	if s.focus == focus_main && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.boostInput.SetText("")
		s.focus = focus_enterBoost
	}

	x, y := ebiten.CursorPosition()
	mouse := v2.V2{X: float64(x), Y: float64(y)}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sysN, ok := s.systemsMonitor.mouseOverSystem(mouse)
		if ok {
			s.showSystemInfo(sysN)
		}
	}

	Data.EngiData.Emissions = CalculateEmissions(Data.Galaxy, Data.PilotData.Ship.Pos)
	Data.EngiData.BSPDegrade = CalculateBSPDegrade(s.ranma)
	CalculateCounters(dt)
	s.CalculateLocalCounters()

	select {
	case <-s.tick:
		if !Data.NaviData.IsOrbiting {
			s.procTick()
		}
	default:
	}

	s.checkForWormHole()

	s.systemsMonitor.update(dt, s.ranma)
}

func (s *engiScene) Draw(image *ebiten.Image) {
	defer LogFunc("engiScene.Draw")()
	Q := s.q
	Q.Clear()

	if s.focus == focus_enterBoost {
		s.q.Append(s.boostInput)
	}
	if s.wrongBoostT > 0 {
		wrongBoostMsg := graph.NewText("WRONG BOOST!", Fonts[Face_cap], colornames.Darkred)
		wrongBoostMsg.SetPosPivot(graph.ScrP(0.5, 0), graph.TopMiddle())
		s.q.Add(wrongBoostMsg, graph.Z_HUD)
	}
	Q.Append(s.systemsMonitor)

	Q.Run(image)
}

func (s *engiScene) OnCommand(command string) {
	switch command {
	case "GDmgHard":
		s.doAZDamage(DEFVAL.HardGDmgRepeats, DEFVAL.HardGDmg)
	case "GDmgMedium":
		s.doAZDamage(DEFVAL.MediumGDmgRepeats, DEFVAL.MediumGDmg)
	default:

	}
}

func (*engiScene) Destroy() {
}

func (s *engiScene) showSystemInfo(n int) {
	//fixme
	log.Println("show system info #", n)
}

func (s *engiScene) procTick() {
	s.checkDamage()
	s.checkMedicine()
}

func (s *engiScene) checkForWormHole() {
	if Data.EngiData.Emissions[EMI_WORMHOLE] == 0 {
		return
	}

	target, err := GetWormHoleTarget(Data.State.GalaxyID)
	if err != nil {
		Log(LVL_ERROR, err)
		return
	}

	if target == WarmHoleYouDIE && s.dieTimeout == 0 {
		s.dieTimeout = 2
		ClientLogGame(Client, "ship", "Die by wormhole")
		Client.SendRequest(CMD_GRACEENDDIE)
		return
	}

	//to other system
	state := Data.State
	state.StateID = STATE_cosmo
	state.GalaxyID = target
	Client.RequestNewState(state.Encode(), false)
}

func (s *engiScene) onBoostTextInput(text string, done bool) {
	s.focus = focus_main
	if done {
		ok := s.tryBoost(text)
		if !ok {
			s.wrongBoostT = 2
		}
	}
}
