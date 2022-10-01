package playback

func FindTrackIndex(trackID uint32, tracks []*Track) int {
	for index, track := range tracks {
		if track.ID == trackID {
			return index
		}
	}

	return -1
}
