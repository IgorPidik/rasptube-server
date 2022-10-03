package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"server/playback"
	"server/server"
	"sync"
	"syscall"
)

func getUpdatedState(playback *playback.Playback) *server.PlaybackState {
	trackIndex := playback.State.TrackIndex
	length, _ := playback.Player.VLCPlayer.MediaLength()
	position, _ := playback.Player.VLCPlayer.MediaPosition()
	currentMilliseconds := uint32(position * float32(length))

	return &server.PlaybackState{
		PlaylistID:       playback.Playlist.ID,
		TrackID:          playback.Playlist.Tracks[trackIndex].ID,
		Playing:          playback.State.Playing,
		TrackCurrentTime: currentMilliseconds,
		TrackTotalTime:   uint32(length),
	}
}

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	ctx, cancel := context.WithCancel(context.Background())

	tracks := []*playback.Track{
		{ID: 0, Name: "song 1", Url: "https://www.youtube.com/watch?v=jEUz8rKJclU"},
		{ID: 1, Name: "song 2", Url: "https://www.youtube.com/watch?v=DZ-ei_OfRrI"},
		{ID: 2, Name: "song 3", Url: "https://www.youtube.com/watch?v=PaaZXV1F1EQ"},
	}

	playlist := &playback.Playlist{ID: 1, Tracks: tracks}

	playbackHandler, playbackErr := playback.NewPlayback(playlist)
	if playbackErr != nil {
		log.Fatal(playbackErr)
	}
	defer playbackHandler.Release()
	playbackEvents := playbackHandler.Init(ctx, wg)

	if playErr := playbackHandler.Play(); playErr != nil {
		log.Fatal(playErr)
	}

	initState := &server.PlaybackData{
		Playlist: playlist,
		State: &server.PlaybackState{
			PlaylistID: playlist.ID,
			Playing:    true,
		},
	}

	s := server.NewServer(ctx, initState)
	serverEvents := s.Init(wg)
	defer s.Stop()

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

out:
	for {
		select {
		case e := <-serverEvents:
			switch e.Type {
			case server.PlaybackTogglePlay:
				playbackHandler.TogglePlay()
			case server.PlaybackPlay:
				playbackHandler.Play()
			case server.PlaybackStop:
				playbackHandler.Pause()
			case server.PlaybackNext:
				playbackHandler.Next()
			case server.PlaybackPrev:
				playbackHandler.Prev()
			case server.PlayTrackByID:
				if payload, ok := e.Payload.(server.PlayTrackByIDPayload); ok {
					playbackHandler.PlayTrack(payload.TrackID)
				}
			default:
				log.Fatalf("unhandled server type: %v", e.Type)
			}
			// update state and publish it
			s.PlaybackData.State = getUpdatedState(playbackHandler)
			s.PublishState()

		case e := <-playbackEvents:
			switch e {
			case playback.TrackFinished:
				playbackHandler.Next()
			}
			// update state and publish it
			s.PlaybackData.State = getUpdatedState(playbackHandler)
			s.PublishState()
		case _ = <-termChan:
			log.Println("Shutting down...")
			cancel()
			break out
		}
	}

	wg.Wait()
	log.Println("Exiting!")
}
