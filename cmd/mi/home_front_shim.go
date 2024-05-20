package main

import (
	"fmt"
	"net/http"

	"within.website/x/cmd/mi/models"
)

type HomeFrontShim struct {
	dao *models.DAO
}

func (hfs *HomeFrontShim) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sw, err := hfs.dao.WhoIsFront(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprint(w, sw.Member.Name)
}
