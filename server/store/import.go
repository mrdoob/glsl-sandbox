package store

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

// JSONEffect contains the BSON dump structure.
type JSONEffect struct {
	ID            int           `json:"_id"`
	CreatedAt     JSONDate      `json:"created_at"`
	ImageURL      string        `json:"image_url"`
	ModifiedAt    JSONDate      `json:"modified_at"`
	Parent        int           `json:"parent"`
	ParentVersion int           `json:"parent_version"`
	User          string        `json:"user"`
	Hidden        bool          `json:"hidden"`
	Versions      []JSONVersion `json:"versions"`
}

type JSONVersion struct {
	CreatedAt JSONDate `json:"created_at"`
	Code      string   `json:"code"`
}

type JSONDate struct {
	Date int64 `json:"$date"`
}

func (j JSONDate) Time() time.Time {
	seconds := j.Date / 1000
	milis := j.Date % 1000
	return time.Unix(seconds, milis*1000000)
}

// Convert a BSON struct into Effect.
func Convert(j JSONEffect) Effect {
	e := Effect{
		ID:            j.ID,
		CreatedAt:     j.CreatedAt.Time(),
		ModifiedAt:    j.ModifiedAt.Time(),
		Parent:        j.Parent,
		ParentVersion: j.ParentVersion,
		User:          j.User,
		Hidden:        j.Hidden,
		Versions:      make([]Version, 0, len(j.Versions)),
	}

	if e.Parent < 1 {
		e.Parent = -1
		e.ParentVersion = -1
	}

	for _, v := range j.Versions {
		e.Versions = append(e.Versions, Version{
			CreatedAt: v.CreatedAt.Time(),
			Code:      v.Code,
		})
	}

	return e
}

// Import a BSON dump to a store.
func Import(r io.Reader, s *Effects) error {
	sb := make([]byte, 16*1024*1024) // initial line buffer of 16 Mb
	scanner := bufio.NewScanner(r)
	scanner.Buffer(sb, 64*1024*1024)

	var j JSONEffect
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) < 3 {
			continue
		}

		err := json.Unmarshal(b, &j)
		if err != nil {
			return err
		}

		err = s.AddEffect(Convert(j))
		if err != nil {
			return err
		}
	}

	return nil
}
