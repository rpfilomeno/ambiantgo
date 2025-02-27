package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/getlantern/systray"
)

type SoundPlayer struct {
	sounds          []string
	currentSound    string
	currentStreamer beep.StreamSeekCloser
	format          beep.Format
	isPlaying       bool
	volume          float64
}

func (sp *SoundPlayer) loadSound(filename string) error {
	// Close existing streamer if open
	if sp.currentStreamer != nil {
		sp.currentStreamer.Close()
	}

	// Open new sound file
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return err
	}

	sp.currentStreamer = streamer
	sp.format = format
	sp.currentSound = filename

	return nil
}

func (sp *SoundPlayer) play() error {
	if sp.currentStreamer == nil {
		return fmt.Errorf("no sound loaded")
	}

	// Initialize speaker if not already initialized
	if err := speaker.Init(sp.format.SampleRate, sp.format.SampleRate.N(time.Second/10)); err != nil {
		return err
	}

	// Reset streamer to beginning
	sp.currentStreamer.Seek(0)

	// Create a looping streamer
	loopStreamer := beep.Loop(-1, sp.currentStreamer)

	// Create a volume-controlled streamer
	volumeCtrl := &beep.Ctrl{Streamer: loopStreamer, Paused: false}

	volume := &effects.Volume{
		Streamer: loopStreamer,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	volume.Volume = sp.volume

	speaker.Play(volume)
	sp.isPlaying = true
	return volumeCtrl.Streamer.Err()
}

func (sp *SoundPlayer) pause() {
	speaker.Clear()
	sp.isPlaying = false
}

func (sp *SoundPlayer) setVolume(vol float64) {
	sp.volume = vol
	if sp.isPlaying {

		// Replay with new volume
		sp.pause()
		sp.play()
	}
}

func main() {
	soundPlayer := &SoundPlayer{
		sounds: []string{
			"./sounds/Mountain Stream.mp3",
		},
		volume: 0,
	}

	// Try to load first sound by default
	if len(soundPlayer.sounds) > 0 {
		soundPlayer.loadSound(soundPlayer.sounds[0])
		soundPlayer.setVolume(-2)
		soundPlayer.play()
	}

	systray.Run(func() {
		// Set the icon from ICO file
		systray.SetIcon(loadIcon("ambiantgo.ico"))

		// Create menu items
		mPlay := systray.AddMenuItem("Play", "Play sound")
		mPause := systray.AddMenuItem("Pause", "Pause sound")

		// Volume submenu
		mVolume := systray.AddMenuItem("Volume", "Adjust Volume")
		mVolumeLow := mVolume.AddSubMenuItem("Low", "Set low volume")
		mVolumeMedium := mVolume.AddSubMenuItem("Medium", "Set medium volume")
		mVolumeHigh := mVolume.AddSubMenuItem("High", "Set high volume")

		// Sounds submenu
		mSounds := systray.AddMenuItem("Sounds", "Select Sound")
		soundMenuItems := make([]*systray.MenuItem, len(soundPlayer.sounds))
		for i, sound := range soundPlayer.sounds {
			soundMenuItems[i] = mSounds.AddSubMenuItem(filepath.Base(sound), "Select this sound")
		}

		mQuit := systray.AddMenuItem("Quit", "Quit the app")

		go func() {
			for {
				select {
				case <-mPlay.ClickedCh:
					soundPlayer.play()
				case <-mPause.ClickedCh:
					soundPlayer.pause()
				case <-mVolumeLow.ClickedCh:
					soundPlayer.setVolume(-3)
				case <-mVolumeMedium.ClickedCh:
					soundPlayer.setVolume(-1)
				case <-mVolumeHigh.ClickedCh:
					soundPlayer.setVolume(0)
				case <-mQuit.ClickedCh:
					systray.Quit()
					return
				}

				// Handle sound selection
				for i, item := range soundMenuItems {
					select {
					case <-item.ClickedCh:
						err := soundPlayer.loadSound(soundPlayer.sounds[i])
						if err != nil {
							log.Println("Error loading sound:", err)
						}
						// If currently playing, restart with new sound
						if soundPlayer.isPlaying {
							soundPlayer.play()
						}
					default:
					}
				}
			}
		}()
	}, func() {
		// Cleanup
		if soundPlayer.currentStreamer != nil {
			soundPlayer.currentStreamer.Close()
		}
		speaker.Close()
	})
}

// loadIcon reads an ICO file and returns its byte content
func loadIcon(filename string) []byte {
	// Read the entire ICO file
	iconBytes, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Error loading icon: %v", err)
		return nil
	}
	return iconBytes
}
