// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package democatalog serves a Subsonic-shaped library for demo mode.
// Artist, album, and track names mirror well-known releases for UI previews.
// Audio streams are silent placeholders. Cover art is fetched at runtime
// through internal/metadata (iTunes lookup) and cached in memory.
package democatalog

import (
	"fmt"
	"strings"
	"sync"

	"melovian/internal/brand"
)

const (
	ServerURL     = "fake://melovian-demo"
	Username      = "demo"
	Password      = "demo"
	ServerName    = "Home Library"
	ServerVersion = "1.16.1"
	SubsonicVer   = "1.16.1"
)

type Artist struct {
	ID         string
	Name       string
	CoverArt   string
	AlbumIDs   []string
	Biography  string
	SimilarIDs []string
}

type Album struct {
	ID        string
	Name      string
	ArtistID  string
	Artist    string
	Year      int
	Genre     string
	CoverArt  string
	SongIDs   []string
	CreatedAt string
}

type Song struct {
	ID          string
	Title       string
	AlbumID     string
	Album       string
	ArtistID    string
	Artist      string
	Track       int
	Duration    int
	Year        int
	Genre       string
	CoverArt    string
	BitRate     int
	Starred     bool
	PlayCount   int
	ContentType string
	Suffix      string
}

type Playlist struct {
	ID      string
	Name    string
	Comment string
	SongIDs []string
	Created string
	Changed string
	Public  bool
	Owner   string
}

type Genre struct {
	Name       string
	SongCount  int
	AlbumCount int
}

type Catalog struct {
	Artists   []Artist
	Albums    []Album
	Songs     []Song
	Playlists []Playlist
	Genres    []Genre

	artistByID map[string]*Artist
	albumByID  map[string]*Album
	songByID   map[string]*Song
	plByID     map[string]*Playlist
}

var (
	catalogOnce sync.Once
	catalog     *Catalog
)

func IsFakeURL(serverURL string) bool {
	return strings.HasPrefix(strings.TrimSpace(strings.ToLower(serverURL)), "fake://")
}

func Get() *Catalog {
	catalogOnce.Do(func() {
		catalog = buildCatalog()
	})
	return catalog
}

func Stats() (songs, albums, artists int) {
	c := Get()
	return len(c.Songs), len(c.Albums), len(c.Artists)
}

func (c *Catalog) Artist(id string) (*Artist, bool) {
	a, ok := c.artistByID[id]
	return a, ok
}

func (c *Catalog) Album(id string) (*Album, bool) {
	a, ok := c.albumByID[id]
	return a, ok
}

func (c *Catalog) Song(id string) (*Song, bool) {
	s, ok := c.songByID[id]
	return s, ok
}

func (c *Catalog) Playlist(id string) (*Playlist, bool) {
	p, ok := c.plByID[id]
	return p, ok
}

func (c *Catalog) CoverLabel(id string) (title, subtitle string) {
	if a, ok := c.albumByID[id]; ok {
		return a.Name, a.Artist
	}
	if a, ok := c.artistByID[id]; ok {
		return a.Name, "Artist"
	}
	if s, ok := c.songByID[id]; ok {
		return s.Album, s.Artist
	}
	if p, ok := c.plByID[id]; ok {
		return p.Name, "Playlist"
	}
	return "Library", brand.Name
}

func (c *Catalog) AlbumSongs(albumID string) []Song {
	album, ok := c.albumByID[albumID]
	if !ok {
		return nil
	}
	out := make([]Song, 0, len(album.SongIDs))
	for _, id := range album.SongIDs {
		if s, ok := c.songByID[id]; ok {
			out = append(out, *s)
		}
	}
	return out
}

func (c *Catalog) ArtistAlbums(artistID string) []Album {
	artist, ok := c.artistByID[artistID]
	if !ok {
		return nil
	}
	out := make([]Album, 0, len(artist.AlbumIDs))
	for _, id := range artist.AlbumIDs {
		if a, ok := c.albumByID[id]; ok {
			out = append(out, *a)
		}
	}
	return out
}

func (c *Catalog) SongsByGenre(genre string) []Song {
	out := make([]Song, 0)
	for i := range c.Songs {
		if strings.EqualFold(c.Songs[i].Genre, genre) {
			out = append(out, c.Songs[i])
		}
	}
	return out
}

func (c *Catalog) Search(query string, artistCount, albumCount, songCount int) (artists []Artist, albums []Album, songs []Song) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil, nil
	}
	for i := range c.Artists {
		if strings.Contains(strings.ToLower(c.Artists[i].Name), q) {
			artists = append(artists, c.Artists[i])
			if len(artists) >= artistCount {
				break
			}
		}
	}
	for i := range c.Albums {
		name := strings.ToLower(c.Albums[i].Name + " " + c.Albums[i].Artist)
		if strings.Contains(name, q) {
			albums = append(albums, c.Albums[i])
			if len(albums) >= albumCount {
				break
			}
		}
	}
	for i := range c.Songs {
		name := strings.ToLower(c.Songs[i].Title + " " + c.Songs[i].Artist + " " + c.Songs[i].Album)
		if strings.Contains(name, q) {
			songs = append(songs, c.Songs[i])
			if len(songs) >= songCount {
				break
			}
		}
	}
	return artists, albums, songs
}

type albumSpec struct {
	name   string
	year   int
	genre  string
	tracks []string
}

type artistSpec struct {
	name    string
	bio     string
	albums  []albumSpec
	similar []int
}

func buildCatalog() *Catalog {
	// Top Spotify monthly listeners (Aug 2026) plus Rezz and Fred again..
	specs := []artistSpec{
		{
			name: "Bruno Mars",
			bio:  "Pop and R&B. Top monthly listeners on Spotify in 2026.",
			albums: []albumSpec{
				// Lead with a widely recognized cover for demo screenshots.
				{name: "24K Magic", year: 2016, genre: "R&B", tracks: []string{
					"24K Magic", "Chunky", "Perm", "That's What I Like", "Versace on the Floor",
					"Straight Up & Down", "Calling All My Lovelies", "Finesse", "Too Good to Say Goodbye",
				}},
				{name: "The Romantic", year: 2026, genre: "Pop", tracks: []string{
					"Risk It All", "Cha Cha Cha", "I Just Might", "God Was Showing Off",
					"Why You Wanna Fight?", "On My Soul", "Something Serious", "Nothing Left", "Dance with Me",
				}},
			},
			similar: []int{3, 5},
		},
		{
			name: "Taylor Swift",
			bio:  "Pop songwriter. Among the most-streamed artists worldwide.",
			albums: []albumSpec{
				{name: "The Life of a Showgirl", year: 2025, genre: "Pop", tracks: []string{
					"The Fate of Ophelia", "Elizabeth Taylor", "Opalite", "Father Figure",
					"Eldest Daughter", "Ruin the Friendship", "Actually Romantic", "Wi$h Li$t",
					"Wood", "Cancelled!", "Honey", "The Life of a Showgirl",
				}},
				{name: "Midnights", year: 2022, genre: "Pop", tracks: []string{
					"Lavender Haze", "Maroon", "Anti-Hero", "Snow On The Beach",
					"You're On Your Own, Kid", "Midnight Rain", "Question...?", "Vigilante Shit",
					"Bejeweled", "Labyrinth", "Karma", "Sweet Nothing", "Mastermind",
				}},
			},
			similar: []int{4, 0},
		},
		{
			name: "The Weeknd",
			bio:  "R&B and pop. Blinding Lights-era catalog still dominates charts.",
			albums: []albumSpec{
				{name: "After Hours", year: 2020, genre: "R&B", tracks: []string{
					"Alone Again", "Too Late", "Hardest To Love", "Scared To Live", "Snowchild",
					"Escape From LA", "Heartless", "Faith", "Blinding Lights", "In Your Eyes",
					"Save Your Tears", "Repeat After Me", "After Hours", "Until I Bleed Out",
				}},
				{name: "Dawn FM", year: 2022, genre: "R&B", tracks: []string{
					"Dawn FM", "Gasoline", "How Do I Make You Love Me?", "Take My Breath",
					"Sacrifice", "A Tale by Quincy", "Out of Time", "Here We Go... Again",
					"Best Friends", "Is There Someone Else?", "Starry Eyes", "Every Angel is Terrifying",
					"Don't Break My Heart", "I Heard You're Married", "Less Than Zero", "Phantom Regret by Jim",
				}},
			},
			similar: []int{0, 5},
		},
		{
			name: "Lady Gaga",
			bio:  "Pop. Die With a Smile with Bruno Mars was a defining 2024-2025 global hit.",
			albums: []albumSpec{
				{name: "Mayhem", year: 2025, genre: "Pop", tracks: []string{
					"Disease", "Abracadabra", "Garden of Eden", "Perfect Celebrity",
					"Vanish into You", "Killah", "Zombieboy", "LoveDrug",
					"How Bad Do U Want Me", "Don't Call Tonight", "Shadow of a Man",
					"The Beast", "Blade of Grass", "Die with a Smile",
				}},
			},
			similar: []int{0, 1},
		},
		{
			name: "Billie Eilish",
			bio:  "Alt-pop. Hit Me Hard and Soft kept her near the top of Spotify followers.",
			albums: []albumSpec{
				{name: "Hit Me Hard and Soft", year: 2024, genre: "Alt Pop", tracks: []string{
					"Skinny", "Lunch", "Chihiro", "Birds of a Feather", "Wildflower",
					"The Greatest", "L'Amour Sur Le Bitume", "The Diner", "Bittersuite", "Blue",
				}},
			},
			similar: []int{1, 7},
		},
		{
			name: "Bad Bunny",
			bio:  "Reggaeton and Latin pop. Long-running Spotify streaming leader.",
			albums: []albumSpec{
				{name: "Debí Tirar Más Fotos", year: 2025, genre: "Reggaeton", tracks: []string{
					"Nuevayol", "Voy a Llevarte Pa' PR", "Baile Inolvidable", "DtMF",
					"Weltita", "Velocidad", "La Mudanza", "Pitorro de Coco",
					"Café con Ron", "Baja Pa' Cá", "EoO", "Piña Colada",
				}},
				{name: "Un Verano Sin Ti", year: 2022, genre: "Reggaeton", tracks: []string{
					"Moscow Mule", "Después de la Playa", "Me Porto Bonito", "Tití Me Preguntó",
					"Un Ratito", "Yo No Soy Celoso", "Tarot", "Neverita", "La Corriente",
					"Efecto", "Party", "Ojitos Lindos", "Amsterdam", "Un Verano Sin Ti",
				}},
			},
			similar: []int{2, 0},
		},
		{
			name: "Rezz",
			bio:  "Canadian electronic producer. Mid-tempo bass and dark techno.",
			albums: []albumSpec{
				{name: "As the Pendulum Swings", year: 2025, genre: "Electronic", tracks: []string{
					"Prophecy", "Contorted", "Telepathy", "Running From Yourself", "Glass Veins",
					"Zone", "INCANTATION", "Substance", "HOW I DO IT", "Downward",
					"Upper", "Blue People", "Delirium",
				}},
				{name: "Can You See Me?", year: 2024, genre: "Electronic", tracks: []string{
					"Black Ice", "Dysphoria", "Can You See Me?", "Everywhere, Nowhere",
					"Edge", "Give In To You", "Dark Age", "Blue in the Face",
				}},
			},
			similar: []int{7, 2},
		},
		{
			name: "Fred again..",
			bio:  "One-man UK producer. Actual Life samples and stadium dance sets.",
			albums: []albumSpec{
				{name: "Actual Life 3 (January 1 - September 9 2022)", year: 2022, genre: "Electronic", tracks: []string{
					"January 1st 2022", "Eyelar (Shutters)", "Delilah (Pull Me Out of This)",
					"Kammy (Like I Do)", "Berwyn (All That I Got Is You)", "Bleu (Better with Time)",
					"Nathan (Still Breathing)", "Danielle (Smile on My Face)", "Kelly (End of a Nightmare)",
					"Mustafa (Time to Move You)", "Clara (The Night Is Dark)", "Winnie (End of Me)",
					"September 9th 2022",
				}},
				{name: "USB", year: 2024, genre: "Electronic", tracks: []string{
					"Rumble", "turn on the lights again..", "adore u", "leavemealone",
					"Baby again..", "Jungle", "places to be", "ten",
				}},
			},
			similar: []int{6, 2},
		},
		{
			name: "Ariana Grande",
			bio:  "Pop. Still among Spotify's top monthly listeners in 2026.",
			albums: []albumSpec{
				{name: "eternal sunshine", year: 2024, genre: "Pop", tracks: []string{
					"intro (end of the world)", "bye", "don't wanna break up again", "Saturn Returns Interlude",
					"eternal sunshine", "supernatural", "true story", "the boy is mine",
					"yes, and?", "we can't be friends (wait for your love)", "i wish i hated you", "imperfect for you", "ordinary things",
				}},
				{name: "hate that i made you love me - Single", year: 2026, genre: "Pop", tracks: []string{
					"hate that i made you love me",
				}},
			},
			similar: []int{1, 3},
		},
		{
			name: "Drake",
			bio:  "Hip-hop and R&B. Permanent fixture near the top of Spotify charts.",
			albums: []albumSpec{
				{name: "Iceman", year: 2026, genre: "Hip-Hop", tracks: []string{
					"Janice STFU", "What Did I Miss?", "Which Way", "Meet Your Padre",
					"Dog House", "Somebody Loves Me", "Gimme a Hug", "N 2 Deep",
					"Raining in Houston", "Ice Dance",
				}},
				{name: "For All The Dogs", year: 2023, genre: "Hip-Hop", tracks: []string{
					"Virginia Beach", "Amen", "Calling For You", "Fear Of Heights",
					"Daylight", "First Person Shooter", "IDGAF", "7969 Santa",
					"Slime You Out", "Bahamas Promises", "Tried Our Best", "Rich Baby Daddy",
				}},
			},
			similar: []int{2, 5},
		},
		{
			name: "Ella Langley",
			bio:  "Country. Choosin' Texas spent 19 weeks at No. 1 on the Hot 100 in 2026.",
			albums: []albumSpec{
				{name: "Dandelion", year: 2026, genre: "Country", tracks: []string{
					"Dandelion", "Choosin' Texas", "We Know Us", "Low Lights", "Be Her",
					"You & Me Time", "Loving Life Again", "Bottom Of Your Boots", "Speaking Terms",
					"I Gotta Quit", "Last Call For Us", "Broken", "Somethin' Simple",
				}},
			},
			similar: []int{0, 1},
		},
		{
			name: "Olivia Rodrigo",
			bio:  "Pop-rock. Drop Dead led her third album onto the Hot 100 and Billboard 200.",
			albums: []albumSpec{
				{name: "you seem pretty sad for a girl so in love", year: 2026, genre: "Pop", tracks: []string{
					"drop dead", "stupid song", "the cure", "you seem pretty sad",
					"looking like an angel", "versailles", "serena joy", "unraveled",
				}},
			},
			similar: []int{4, 1},
		},
		{
			name: "Harry Styles",
			bio:  "Pop. Aperture debuted at No. 1 on the Hot 100 in February 2026.",
			albums: []albumSpec{
				{name: "Kiss All the Time. Disco, Occasionally", year: 2026, genre: "Pop", tracks: []string{
					"Aperture", "American Girls", "Ready, Steady, Go!", "Are You Listening Yet?",
					"Taste Back", "The Waiting Game", "Season 2 Weight Loss", "Coming Up Roses",
					"Pop", "Dance No More", "Paint by Numbers", "Carla's Song",
				}},
			},
			similar: []int{1, 0},
		},
		{
			name: "BTS",
			bio:  "K-pop. Swim debuted at No. 1 on the Hot 100. Arirang led the Billboard 200.",
			albums: []albumSpec{
				{name: "ARIRANG", year: 2026, genre: "K-Pop", tracks: []string{
					"SWIM", "Body to Body", "Hooligan", "NORMAL", "2.0", "FYA",
					"they don't know 'bout us", "Like Animals", "HERE 4 U", "Arirang",
				}},
			},
			similar: []int{1, 8},
		},
	}

	c := &Catalog{
		artistByID: map[string]*Artist{},
		albumByID:  map[string]*Album{},
		songByID:   map[string]*Song{},
		plByID:     map[string]*Playlist{},
	}

	genreCounts := map[string]struct{ songs, albums int }{}
	songIdx := 0

	for ai, spec := range specs {
		artistID := fmt.Sprintf("ar-%03d", ai+1)
		artist := Artist{
			ID:        artistID,
			Name:      spec.name,
			CoverArt:  artistID,
			Biography: spec.bio,
		}

		for aj, alb := range spec.albums {
			albumID := fmt.Sprintf("al-%03d-%02d", ai+1, aj+1)
			album := Album{
				ID:        albumID,
				Name:      alb.name,
				ArtistID:  artistID,
				Artist:    spec.name,
				Year:      alb.year,
				Genre:     alb.genre,
				CoverArt:  albumID,
				CreatedAt: fmt.Sprintf("%d-06-15T12:00:00Z", alb.year),
			}

			for ti, title := range alb.tracks {
				songIdx++
				songID := fmt.Sprintf("tr-%04d", songIdx)
				dur := 185 + ((songIdx * 19) % 155)
				playCount := (songIdx * 3) % 40
				if alb.year >= 2025 {
					playCount = 55 + (songIdx % 35)
				}
				song := Song{
					ID:          songID,
					Title:       title,
					AlbumID:     albumID,
					Album:       alb.name,
					ArtistID:    artistID,
					Artist:      spec.name,
					Track:       ti + 1,
					Duration:    dur,
					Year:        alb.year,
					Genre:       alb.genre,
					CoverArt:    albumID,
					BitRate:     320,
					ContentType: "audio/mpeg",
					Suffix:      "mp3",
					PlayCount:   playCount,
					Starred:     songIdx%7 == 0 || songIdx%11 == 0,
				}
				c.Songs = append(c.Songs, song)
				album.SongIDs = append(album.SongIDs, songID)
				gc := genreCounts[alb.genre]
				gc.songs++
				genreCounts[alb.genre] = gc
			}

			gc := genreCounts[alb.genre]
			gc.albums++
			genreCounts[alb.genre] = gc

			artist.AlbumIDs = append(artist.AlbumIDs, albumID)
			c.Albums = append(c.Albums, album)
		}

		c.Artists = append(c.Artists, artist)
	}

	for i := range c.Artists {
		for _, si := range specs[i].similar {
			if si >= 0 && si < len(c.Artists) {
				c.Artists[i].SimilarIDs = append(c.Artists[i].SimilarIDs, c.Artists[si].ID)
			}
		}
	}

	for i := range c.Artists {
		c.artistByID[c.Artists[i].ID] = &c.Artists[i]
	}
	for i := range c.Albums {
		c.albumByID[c.Albums[i].ID] = &c.Albums[i]
	}
	for i := range c.Songs {
		c.songByID[c.Songs[i].ID] = &c.Songs[i]
	}

	for name, counts := range genreCounts {
		c.Genres = append(c.Genres, Genre{
			Name:       name,
			SongCount:  counts.songs,
			AlbumCount: counts.albums,
		})
	}

	c.Playlists = []Playlist{
		{
			ID: "pl-hits", Name: "2026 Chart Stack", Comment: "Hot 100 leaders from 2026",
			SongIDs: pickByTitle(c, []string{
				"Choosin' Texas", "I Just Might", "Aperture", "DtMF", "Opalite",
				"SWIM", "drop dead", "Janice STFU", "hate that i made you love me", "Be Her",
			}),
			Created: "2026-01-10T10:00:00Z", Changed: "2026-08-01T10:00:00Z", Public: true, Owner: "demo",
		},
		{
			ID: "pl-bass", Name: "Rezz + Fred", Comment: "Electronic late nights",
			SongIDs: pickByTitle(c, []string{
				"Prophecy", "Contorted", "Telepathy", "Running From Yourself",
				"Delilah (Pull Me Out of This)", "Kammy (Like I Do)", "Rumble", "adore u",
			}),
			Created: "2025-11-02T22:00:00Z", Changed: "2026-07-12T22:00:00Z", Public: true, Owner: "demo",
		},
		{
			ID: "pl-pop", Name: "Pop Rotation", Comment: "Bruno, Taylor, Harry, Olivia, Ariana",
			SongIDs: pickByTitle(c, []string{
				"I Just Might", "Aperture", "drop dead", "Die with a Smile",
				"Birds of a Feather", "Anti-Hero", "yes, and?", "Abracadabra",
			}),
			Created: "2024-03-17T09:00:00Z", Changed: "2026-06-20T09:00:00Z", Public: false, Owner: "demo",
		},
		{
			ID: "pl-latin", Name: "Un Verano Cuts", Comment: "Bad Bunny essentials",
			SongIDs: pickByTitle(c, []string{
				"DtMF", "Me Porto Bonito", "Tití Me Preguntó", "Ojitos Lindos", "Efecto", "Moscow Mule",
			}),
			Created: "2022-08-08T18:00:00Z", Changed: "2026-05-04T18:00:00Z", Public: true, Owner: "demo",
		},
		{
			ID: "pl-country", Name: "Choosin' Country", Comment: "Ella Langley Dandelion era",
			SongIDs: pickByTitle(c, []string{
				"Choosin' Texas", "Be Her", "Dandelion", "Loving Life Again", "We Know Us",
			}),
			Created: "2026-04-10T12:00:00Z", Changed: "2026-08-20T12:00:00Z", Public: true, Owner: "demo",
		},
	}
	for i := range c.Playlists {
		c.plByID[c.Playlists[i].ID] = &c.Playlists[i]
	}

	return c
}

func pickByTitle(c *Catalog, titles []string) []string {
	out := make([]string, 0, len(titles))
	for _, title := range titles {
		for i := range c.Songs {
			if strings.EqualFold(c.Songs[i].Title, title) {
				out = append(out, c.Songs[i].ID)
				break
			}
		}
	}
	return out
}
