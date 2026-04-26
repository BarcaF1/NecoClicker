package macro

type ActionType string

const (
	ActionClick ActionType = "click"
	ActionMove  ActionType = "move"
	ActionDelay ActionType = "delay"
)

type MouseButton string

const (
	BtnLeft   MouseButton = "left"
	BtnRight  MouseButton = "right"
	BtnMiddle MouseButton = "middle"
	BtnX1     MouseButton = "x1" // боковая "вперёд" (Mouse4)
	BtnX2     MouseButton = "x2" // боковая "назад"  (Mouse5)
)

// Action — единичный шаг макроса.
type Action struct {
	Type       ActionType  `json:"type"`
	X          int         `json:"x,omitempty"`
	Y          int         `json:"y,omitempty"`
	Relative   bool        `json:"relative,omitempty"`
	UseCurrent bool        `json:"use_current,omitempty"`
	Button     MouseButton `json:"button,omitempty"`
	DelayMs    int         `json:"delay_ms,omitempty"`
}

type Chain struct {
	Name    string   `json:"name"`
	Hotkey  string   `json:"hotkey,omitempty"`
	Loops   int      `json:"loops"`
	Actions []Action `json:"actions"`
}

// SimpleConfig — настройки одного "профиля" простого кликера.
// Один Config может хранить несколько таких профилей (см. Config.Profiles).
//
// IntervalMs — float, поддерживает доли миллисекунды (например, 0.5).
// Значение 0 = "максимальная скорость" (tight-loop без сна).
type SimpleConfig struct {
	Name       string      `json:"name"`
	Button     MouseButton `json:"button"`
	IntervalMs float64     `json:"interval_ms"`
	UseCurrent bool        `json:"use_current"`
	X          int         `json:"x"`
	Y          int         `json:"y"`
	Hotkey     string      `json:"hotkey"`
}

type Config struct {
	Profiles []SimpleConfig `json:"profiles"`
	Active   int            `json:"active"`
	Chains   []Chain        `json:"chains"`
	Theme    string         `json:"theme"`

	// Legacy: одиночный профиль из v1.0. На загрузке мигрируется в Profiles[0].
	LegacySimple *SimpleConfig `json:"simple,omitempty"`
}

// ActiveProfile возвращает текущий активный профиль (или дефолтный если ничего нет).
func (c *Config) ActiveProfile() SimpleConfig {
	if len(c.Profiles) == 0 {
		return DefaultProfile()
	}
	idx := c.Active
	if idx < 0 || idx >= len(c.Profiles) {
		idx = 0
	}
	return c.Profiles[idx]
}

func DefaultProfile() SimpleConfig {
	return SimpleConfig{
		Name:       "Default",
		Button:     BtnLeft,
		IntervalMs: 100,
		UseCurrent: true,
		Hotkey:     "F6",
	}
}

func DefaultConfig() *Config {
	return &Config{
		Profiles: []SimpleConfig{DefaultProfile()},
		Active:   0,
		Chains:   []Chain{},
		Theme:    "", // пусто = фронт сам подберёт по системной (auto)
	}
}

// migrate выполняет апгрейд старого формата (v1.0) к новому (v1.1).
func (c *Config) migrate() {
	if c.LegacySimple != nil {
		p := *c.LegacySimple
		if p.Name == "" {
			p.Name = "Default"
		}
		if len(c.Profiles) == 0 {
			c.Profiles = []SimpleConfig{p}
			c.Active = 0
		}
		c.LegacySimple = nil
	}
	if len(c.Profiles) == 0 {
		c.Profiles = []SimpleConfig{DefaultProfile()}
		c.Active = 0
	}
	if c.Active < 0 || c.Active >= len(c.Profiles) {
		c.Active = 0
	}
	if c.Chains == nil {
		c.Chains = []Chain{}
	}
	// гарантируем имя у каждого профиля
	for i := range c.Profiles {
		if c.Profiles[i].Name == "" {
			c.Profiles[i].Name = "Profile " + itoa(i+1)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
