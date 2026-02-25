package ui

import "charm.land/lipgloss/v2"

const splashArt = `
                          ░▒▒▒▒▒▒░                          
                   ░▓█████▓▓▓▓▓▓▓▓▓███▓▓░                   
                ▓██▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓██▓                
             ▒█▓▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓█▒             
           ▓█▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓█▓           
         ▒█▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓█▒         
        ▓▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓        
       █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓▓▓▒▒▒▒▒▒▒▓█       
      █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓█████  ▓▒▒▒▒▒▒▒▒▓█      
     █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓████▒       █▒▒▒▒▒▒▒▒▒▓█     
    ▒▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓████▒            █▒▒▒▒▒▒▒▒▒▒▓▒    
    █▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓████▒        ▒█      ▓▓▒▒▒▒▒▒▒▒▒▒▒█    
   ░█▒▒▒▒▒▒▒▒▒▒▓████▒          ░▓▓        █▒▒▒▒▒▒▒▒▒▒▒▒▓░   
   ▓▓▒▒▒▒▒▒▒▒▒▓▒            ░▓▓▓         ▓▓▒▒▒▒▒▒▒▒▒▒▒▒▓▓   
   ▓▓▒▒▒▒▒▒▒▒▒▓          ▒▒▒▒▓           █▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓   
   ▒▓▒▒▒▒▒▒▒▒▒▓█████▒  ▒▒▒▒▒             █▒▒▒▒▒▒▒▒▒▒▒▒▒▓▒   
   ░█▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓█                  █▓▒▒▒▒▒▒▒▒▒▒▒▒▒█    
    █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓█  ▒▓▒           ░█▒▒▒▒▒▒▒▒▒▒▒▒▒▓█    
    ▒█▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▒ ▒░░░▓▒        █▓▒▒▒▒▒▒▒▒▒▒▒▒▒█▒    
     █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒█   ▒▓▓███░     █▒▒▒▒▒▒▒▒▒▒▒▒▒▓█     
      █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓▓▓▓▒▒▒▒▓██▓  █▓▒▒▒▒▒▒▒▒▒▒▒▒▓█      
       █▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓██▓▒▒▒▒▒▒▒▒▒▒▒▒██       
        ▓█▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓█▓        
          ██▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓██          
           ▒██▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓██▒           
              ███▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓███              
                ░████▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓████░                
                    ▒██████████████████▒                    
`

// SplashModel renders a centered splash overlay on startup.
// It stays visible for at least the minimum duration even if
// the connection becomes ready sooner.
type SplashModel struct {
	visible       bool
	timerDone     bool
	connReady     bool
	width, height int
}

// NewSplashModel creates a visible splash.
func NewSplashModel() SplashModel {
	return SplashModel{visible: true}
}

// SetSize updates the terminal dimensions for centering.
func (s SplashModel) SetSize(w, h int) SplashModel {
	s.width = w
	s.height = h
	return s
}

// IsVisible reports whether the splash is still showing.
func (s SplashModel) IsVisible() bool {
	return s.visible
}

// TimerDone marks the minimum display duration as elapsed.
// The splash dismisses only once both the timer is done and
// the connection is ready (or the timer alone suffices).
func (s SplashModel) TimerDone() SplashModel {
	s.timerDone = true
	if s.connReady {
		s.visible = false
	}
	return s
}

// ConnReady marks the connection as established.
// The splash dismisses only if the minimum timer has also elapsed.
func (s SplashModel) ConnReady() SplashModel {
	s.connReady = true
	if s.timerDone {
		s.visible = false
	}
	return s
}

// View renders the splash box (without full-screen placement).
// Use BoxOffset to get the X/Y for centering via the Layer API.
func (s SplashModel) View() string {
	if !s.visible || s.width == 0 || s.height == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 3).
		Margin(2, 4).
		BorderForegroundBlend(rainbowBlend...)

	return style.Render(splashArt)
}

// BoxOffset returns the (x, y) needed to center the splash box
// within the terminal dimensions.
func (s SplashModel) BoxOffset() (int, int) {
	box := s.View()
	bw := lipgloss.Width(box)
	bh := lipgloss.Height(box)
	x := (s.width - bw) / 2
	y := (s.height - bh) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}
