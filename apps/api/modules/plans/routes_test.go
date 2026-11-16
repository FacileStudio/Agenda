package plans

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestRoutes(t *testing.T) {
	h := newHarness(t)
	user := h.owner.ID
	router := routerAs(h.service, &user)
	plan := h.propose(h.block(9, 10), h.block(11, 12))
	path := "/plans/" + strconv.FormatInt(plan.ID, 10)

	rec := serve(router, http.MethodGet, "/plans/", "")
	var list []PlanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != http.StatusOK || len(list) != 1 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec := serve(router, http.MethodGet, path, ""); rec.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := serve(router, http.MethodGet, "/plans/abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("a bad id is 400, got %d", rec.Code)
	}
	if rec := serve(router, http.MethodPost, path+"/apply", `{"block_ids":[1],"unknown":true}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown fields are 400, got %d", rec.Code)
	}

	other := h.other.ID
	stranger := routerAs(h.service, &other)
	for _, call := range []struct{ method, path string }{
		{http.MethodGet, path}, {http.MethodPost, path + "/apply"}, {http.MethodPost, path + "/dismiss"},
	} {
		if rec := serve(stranger, call.method, call.path, ""); rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s as another person: want 404, got %d", call.method, call.path, rec.Code)
		}
	}

	body := `{"block_ids":[` + strconv.FormatInt(plan.Blocks[1].ID, 10) + `]}`
	rec = serve(router, http.MethodPost, path+"/apply", body)
	var applied ApplyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &applied); err != nil || rec.Code != http.StatusOK || len(applied.Events) != 1 || applied.Plan.State != "applied" {
		t.Fatalf("apply a subset: %d %s", rec.Code, rec.Body)
	}
	if rec := serve(router, http.MethodPost, path+"/apply", ""); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("a second apply is 412, got %d", rec.Code)
	}

	next := h.propose(h.block(14, 15))
	rec = serve(router, http.MethodPost, "/plans/"+strconv.FormatInt(next.ID, 10)+"/apply", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("an empty body applies every block, got %d %s", rec.Code, rec.Body)
	}
	last := h.propose(h.block(16, 17))
	if rec := serve(router, http.MethodPost, "/plans/"+strconv.FormatInt(last.ID, 10)+"/dismiss", ""); rec.Code != http.StatusOK {
		t.Fatalf("dismiss: %d %s", rec.Code, rec.Body)
	}
}
