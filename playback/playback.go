package playback

import (
	"errors"
	player "server/player"

	vlc "github.com/adrg/libvlc-go/v3"
)

type Track struct {
	ID   uint32
	Name string
	Url  string
}

type Playlist struct {
	ID     uint32
	Tracks []*Track
}

type state struct {
	TrackIndex int
	Playing    bool
}

type Playback struct {
	Player   *player.YoutubePlayer
	Playlist *Playlist
	State    *state
}

type PlaybackEvent uint

const (
	TrackFinished = iota
	TrackPositionChanged
)

func NewPlayback(p *Playlist) (*Playback, error) {
	ytPlayer, ytPlayerErr := player.NewYoutubePlayer(nil, nil)
	if ytPlayerErr != nil {
		return nil, ytPlayerErr
	}

	return &Playback{
		Player:   ytPlayer,
		Playlist: p,
		State: &state{
			TrackIndex: 0,
			Playing:    false,
		},
	}, nil
}

func (p *Playback) Init() <-chan PlaybackEvent {
	ch := make(chan PlaybackEvent)

	p.Player.VLCPlayerEventManager.Attach(vlc.MediaPlayerEndReached, func(event vlc.Event, i interface{}) {
		ch <- TrackFinished
	}, nil)

	p.Player.VLCPlayerEventManager.Attach(vlc.MediaPlayerPositionChanged, func(event vlc.Event, i interface{}) {
		ch <- TrackPositionChanged
	}, nil)

	return ch
}

func (p *Playback) Play() error {
	trackIndex := p.State.TrackIndex % len(p.Playlist.Tracks)
	if err := p.Player.Play(p.Playlist.Tracks[trackIndex].Url); err != nil {
		return err
	}
	p.State.Playing = true
	return nil
}

func (p *Playback) Pause() {
	p.Player.SetPause(true)
	p.State.Playing = false
}

func (p *Playback) TogglePlay() {
	p.State.Playing = !p.State.Playing
	p.Player.SetPause(!p.State.Playing)
}

func (p *Playback) Next() error {
	p.State.TrackIndex = (p.State.TrackIndex + 1) % len(p.Playlist.Tracks)
	return p.Play()
}

func (p *Playback) Prev() error {
	p.State.TrackIndex = (p.State.TrackIndex - 1) % len(p.Playlist.Tracks)
	return p.Play()
}

func (p *Playback) PlayTrack(trackID uint32) error {
	index := FindTrackIndex(trackID, p.Playlist.Tracks)
	if index == -1 {
		return errors.New("invalid track id")
	}
	p.State.TrackIndex = index
	return p.Play()
}

func (p *Playback) Release() {
	p.Player.Release()
}
