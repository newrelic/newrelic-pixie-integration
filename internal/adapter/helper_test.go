package adapter

import (
	"testing"
)

func TestShouldFilterEmtpy(t *testing.T) {
	rh, err := NewResourceHelper("", "")
	if err != nil {
		t.Errorf("empty excludes should not return error")
	}

	filter := rh.shouldFilter("what", "ever")
	if filter == true {
		t.Errorf("empty excludes should not trigger filter")
	}
}

func TestShouldFilterAll(t *testing.T) {
	rh, err := NewResourceHelper("", ".*")
	if err != nil {
		t.Errorf("builder should not return error")
	}

	filter := rh.shouldFilter("what", "ever")
	if filter == false {
		t.Errorf("filter should always trigger")
	}

	rh, err = NewResourceHelper(".*", "")
	if err != nil {
		t.Errorf("builder should not return error")
	}

	filter = rh.shouldFilter("what", "ever")
	if filter == false {
		t.Errorf("filter should always trigger")
	}
}

func TestShouldFilterNamespace(t *testing.T) {
	rh, err := NewResourceHelper("", "^new$")
	if err != nil {
		t.Errorf("builder should not return error")
	}

	filter := rh.shouldFilter("newrelic", "")
	if filter == true {
		t.Errorf("partial match should not trigger filter")
	}

	filter = rh.shouldFilter("new", "")
	if filter == false {
		t.Errorf("full match should trigger filter")
	}
}

func TestShouldFilterPod(t *testing.T) {
	rh, err := NewResourceHelper("^new$", "")
	if err != nil {
		t.Errorf("builder should not return error")
	}

	filter := rh.shouldFilter("", "newrelic")
	if filter == true {
		t.Errorf("partial match should not trigger filter")
	}

	filter = rh.shouldFilter("", "new")
	if filter == false {
		t.Errorf("full match should trigger filter")
	}
}

func TestShouldFilterBoth(t *testing.T) {
	rh, err := NewResourceHelper("new", "relic")
	if err != nil {
		t.Errorf("builder should not return error")
	}

	filter := rh.shouldFilter("new", "relic")
	if filter == true {
		t.Errorf("should not trigger filter")
	}

	filter = rh.shouldFilter("relic", "new")
	if filter == false {
		t.Errorf("should trigger filter")
	}

	filter = rh.shouldFilter("relic", "")
	if filter == false {
		t.Errorf("should trigger filter")
	}

	filter = rh.shouldFilter("", "new")
	if filter == false {
		t.Errorf("should trigger filter")
	}
}
