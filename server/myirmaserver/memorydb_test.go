package myirmaserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryDBUserManagement(t *testing.T) {
	db := &MyirmaMemoryDB{
		UserData: map[string]MemoryUserData{
			"testuser": MemoryUserData{
				ID:         15,
				LastActive: time.Unix(0, 0),
			},
		},
		VerifyEmailTokens: map[string]int64{
			"testtoken": 15,
		},
	}

	id, err := db.GetUserID("testuser")
	assert.NoError(t, err)
	assert.Equal(t, int64(15), id)

	id, err = db.VerifyEmailToken("testtoken")
	assert.NoError(t, err)
	assert.Equal(t, int64(15), id)

	_, err = db.VerifyEmailToken("testtoken")
	assert.Error(t, err)

	_, err = db.GetUserID("DNE")
	assert.Error(t, err)

	err = db.SetSeen(15)
	assert.NoError(t, err)

	err = db.SetSeen(123456)
	assert.Error(t, err)

	assert.NotEqual(t, time.Unix(0, 0), db.UserData["testuser"].LastActive)

	err = db.RemoveUser(15)
	assert.NoError(t, err)

	_, err = db.GetUserID("testuser")
	assert.Error(t, err)

	err = db.RemoveUser(15)
	assert.Error(t, err)
}

func TestMemoryDBLoginToken(t *testing.T) {
	db := &MyirmaMemoryDB{
		UserData: map[string]MemoryUserData{
			"testuser": MemoryUserData{
				ID:         15,
				LastActive: time.Unix(0, 0),
				Email:      []string{"test@test.com"},
			},
			"noemail": MemoryUserData{
				ID:         17,
				LastActive: time.Unix(0, 0),
			},
		},
		LoginEmailTokens: map[string]string{},
	}

	err := db.AddEmailLoginToken("test2@test.com", "test2token")
	assert.Error(t, err)

	err = db.AddEmailLoginToken("test@test.com", "testtoken")
	require.NoError(t, err)

	cand, err := db.LoginTokenGetCandidates("testtoken")
	assert.NoError(t, err)
	assert.Equal(t, []LoginCandidate{LoginCandidate{Username: "testuser", LastActive: 0}}, cand)

	_, err = db.LoginTokenGetCandidates("DNE")
	assert.Error(t, err)

	email, err := db.LoginTokenGetEmail("testtoken")
	assert.NoError(t, err)
	assert.Equal(t, "test@test.com", email)

	_, err = db.LoginTokenGetEmail("DNE")
	assert.Error(t, err)

	_, err = db.TryUserLoginToken("testtoken", "DNE")
	assert.Error(t, err)

	ok, err := db.TryUserLoginToken("testtoken", "noemail")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = db.TryUserLoginToken("testtoken", "testuser")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = db.TryUserLoginToken("testtoken", "testuser")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestMemoryDBUserInfo(t *testing.T) {
	db := &MyirmaMemoryDB{
		UserData: map[string]MemoryUserData{
			"testuser": MemoryUserData{
				ID:         15,
				LastActive: time.Unix(15, 0),
				Email:      []string{"test@test.com"},
				LogEntries: []LogEntry{
					LogEntry{
						Timestamp: 110,
						Event:     "test",
						Param:     "",
					},
					LogEntry{
						Timestamp: 120,
						Event:     "test2",
						Param:     "15",
					},
				},
			},
			"noemail": MemoryUserData{
				ID:         17,
				LastActive: time.Unix(20, 0),
			},
		},
	}

	info, err := db.GetUserInformation(15)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", info.Username)
	assert.Equal(t, []string{"test@test.com"}, info.Emails)

	info, err = db.GetUserInformation(17)
	assert.NoError(t, err)
	assert.Equal(t, "noemail", info.Username)
	assert.Equal(t, []string(nil), info.Emails)

	_, err = db.GetUserInformation(1231)
	assert.Error(t, err)

	entries, err := db.GetLogs(15, 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, []LogEntry{
		LogEntry{
			Timestamp: 110,
			Event:     "test",
			Param:     "",
		},
		LogEntry{
			Timestamp: 120,
			Event:     "test2",
			Param:     "15",
		},
	}, entries)

	entries, err = db.GetLogs(15, 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(entries))

	entries, err = db.GetLogs(15, 1, 15)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(entries))

	entries, err = db.GetLogs(15, 100, 20)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(entries))

	_, err = db.GetLogs(20, 100, 20)
	assert.Error(t, err)

	err = db.AddEmail(17, "test@test.com")
	assert.NoError(t, err)

	info, err = db.GetUserInformation(17)
	assert.NoError(t, err)
	assert.Equal(t, []string{"test@test.com"}, info.Emails)

	err = db.AddEmail(20, "bla@bla.com")
	assert.Error(t, err)

	err = db.RemoveEmail(17, "test@test.com")
	assert.NoError(t, err)

	info, err = db.GetUserInformation(17)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(info.Emails))

	err = db.RemoveEmail(17, "bla@bla.com")
	assert.NoError(t, err)

	err = db.RemoveEmail(20, "bl@bla.com")
	assert.Error(t, err)
}
