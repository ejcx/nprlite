package nprlite

import (
	"fmt"
	"net/http"
	"testing"
)

func TestArticleFetchApples(t *testing.T) {
	resp, err := http.Get("http://text.npr.org/s.php?sId=565664321")
	if err != nil {
		t.Errorf("Couldn't fetch apples: %s", err)
	}
	body, err := parsearticle(resp.Body)
	if err != nil {
		t.Errorf("Couldn't parse apples: %s", err)
	}
	fmt.Println(body)
}

func TestArticleFetchTrump1(t *testing.T) {
	resp, err := http.Get("http://text.npr.org/s.php?sId=567843266")
	if err != nil {
		t.Errorf("Couldn't fetch trump1: %s", err)
	}
	body, err := parsearticle(resp.Body)
	if err != nil {
		t.Errorf("Couldn't parse trump1: %s", err)
	}
	fmt.Println(body)
}
