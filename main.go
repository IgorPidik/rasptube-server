package main

import (
	"log"
	"server/playback"
	"server/server"
)

func main() {
	tracks := []*playback.Track{
		{ID: 0, Name: "song 1", Url: "https://www.youtube.com/watch?v=jEUz8rKJclU"},
		{ID: 1, Name: "song 2", Url: "https://www.youtube.com/watch?v=DZ-ei_OfRrI"},
		{ID: 2, Name: "song 3", Url: "https://www.youtube.com/watch?v=PaaZXV1F1EQ"},
	}

	playlist := &playback.Playlist{ID: 1, Tracks: tracks}

	playback, playbackErr := playback.NewPlayback(playlist)
	if playbackErr != nil {
		log.Fatal(playbackErr)
	}
	defer playback.Release()

	server := server.NewServer(playback)

	if playErr := playback.Play(); playErr != nil {
		log.Fatal(playErr)
	}

	server.Start()
	defer server.Stop()
}
