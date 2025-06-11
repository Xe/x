package importer

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
	"within.website/x/cmd/mi/models"
	pb "within.website/x/gen/within/website/x/mi/v1"
)

const timeLayout = "2006-01-02 15:04:05"

func New(dao *models.DAO) *Importer {
	return &Importer{db: dao.DB()}
}

type Importer struct {
	db *gorm.DB
}

func (i *Importer) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/.within/mi/import/switches", i.importSwitches)
	mux.HandleFunc("/.within/mi/import/members", i.importMembers)
}

func (i *Importer) importSwitches(w http.ResponseWriter, r *http.Request) {
	rdr := csv.NewReader(r.Body)
	defer r.Body.Close()

	tx := i.db.Begin()

	for {
		row, err := rdr.Read()
		if err != nil {
			break
		}

		if len(row) != 4 {
			slog.Error("invalid row", "row", row)
			continue
		}

		id := row[0]
		memberIDStr := row[1]
		startedAtStr := row[2]
		endedAtStr := row[3]

		memberID, err := strconv.Atoi(memberIDStr)
		if err != nil {
			slog.Error("failed to parse member ID", "err", err)
			continue
		}

		startedAt, err := time.Parse(timeLayout, startedAtStr)
		if err != nil {
			slog.Error("failed to parse started at", "err", err)
			continue
		}

		var endedAt *time.Time
		if endedAtStr != "" {
			endedAtTime, err := time.Parse(timeLayout, endedAtStr)
			if err != nil {
				slog.Error("failed to parse ended at", "err", err, "endedAtStr", endedAtStr)
				continue
			}

			endedAt = &endedAtTime
		}

		var member models.Member
		if err := tx.Where("id = ?", memberID).First(&member).Error; err != nil {
			slog.Error("failed to find member", "err", err, "memberID", memberID)
			continue
		}

		sw := models.Switch{
			ID:       id,
			MemberID: memberID,
			Model: gorm.Model{
				CreatedAt: startedAt,
			},
			EndedAt: endedAt,
		}

		if err := tx.Save(&sw).Error; err != nil {
			slog.Error("failed to save switch", "err", err)
			continue
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("failed to commit transaction", "err", err)
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (i *Importer) importMembers(w http.ResponseWriter, r *http.Request) {
	var members []pb.Member
	if err := json.NewDecoder(r.Body).Decode(&members); err != nil {
		slog.Error("failed to decode members", "err", err)
		http.Error(w, "failed to decode members", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	tx := i.db.Begin()

	for _, m := range members {
		member := models.Member{
			ID:        int(m.Id),
			Name:      m.Name,
			AvatarURL: m.AvatarUrl,
		}

		if err := tx.Exec("INSERT INTO members (id, name, avatar_url) VALUES (?, ?, ?) ON CONFLICT (id) DO UPDATE SET name = ?, avatar_url = ?", member.ID, member.Name, member.AvatarURL, member.Name, member.AvatarURL).Error; err != nil {
			slog.Error("failed to save member", "err", err)
			http.Error(w, "failed to save member", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("failed to commit transaction", "err", err)
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}
}
