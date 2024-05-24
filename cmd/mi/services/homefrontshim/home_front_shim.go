package homefrontshim

import (
	"fmt"
	"net/http"

	"within.website/x/cmd/mi/models"
)

func New(dao *models.DAO) *HomeFrontShim {
	return &HomeFrontShim{dao: dao}
}

type HomeFrontShim struct {
	dao *models.DAO
}

func (hfs *HomeFrontShim) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sw, err := hfs.dao.WhoIsFront(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, sw.Member.Name)
}
