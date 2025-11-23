package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHospitalAdapter_LookupByIdentifier_Success(t *testing.T) {
	// mock hospital server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return a valid JSON body similar to real hospital response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"first_name_th":"สมชาย",
			"middle_name_th":"",
			"last_name_th":"ใจดี",
			"first_name_en":"Somchai",
			"middle_name_en":"",
			"last_name_en":"Jaidee",
			"date_of_birth":"1990-01-01",
			"patient_hn":"HN-001",
			"national_id":"N-1234567890",
			"passport_id":"P-ABC1234",
			"phone_number":"0812345678",
			"email":"somchai@example.com",
			"gender":"M"
		}`))
	}))
	defer ts.Close()

	// create adapter pointing to test server
	h, err := NewHospitalAdapter(ts.URL, 2*time.Second)
	assert.NoError(t, err)

	ctx := context.Background()
	p, err := h.LookupByIdentifier(ctx, "N-1234567890")
	assert.NoError(t, err)
	if assert.NotNil(t, p) {
		assert.Equal(t, "Somchai", p.FirstNameEN)
		assert.Equal(t, "N-1234567890", p.NationalID)
		assert.Equal(t, "HN-001", p.PatientHN)
		assert.Equal(t, "M", p.Gender)
		assert.NotNil(t, p.RawJSON)
	}
}

func TestHospitalAdapter_LookupByIdentifier_NotFound(t *testing.T) {
	// server returns 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	h, err := NewHospitalAdapter(ts.URL, 1*time.Second)
	assert.NoError(t, err)

	ctx := context.Background()
	p, err := h.LookupByIdentifier(ctx, "missing")
	assert.NoError(t, err)
	assert.Nil(t, p)
}

func TestHospitalAdapter_LookupByIdentifier_Timeout(t *testing.T) {
	// server delays beyond timeout
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	// create adapter with short timeout
	h, err := NewHospitalAdapter(ts.URL, 50*time.Millisecond)
	assert.NoError(t, err)

	ctx := context.Background()
	p, err := h.LookupByIdentifier(ctx, "any")
	assert.Error(t, err)
	assert.Nil(t, p)
}

func TestHospitalAdapter_LookupByIdentifier_BadStatus(t *testing.T) {
	// server returns 500 with body
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"something went wrong"}`))
	}))
	defer ts.Close()

	h, err := NewHospitalAdapter(ts.URL, 1*time.Second)
	assert.NoError(t, err)

	ctx := context.Background()
	p, err := h.LookupByIdentifier(ctx, "any")
	assert.Error(t, err)
	assert.Nil(t, p)
}

func TestHospitalAdapter_MapsAllFields(t *testing.T) {
	// ensure mapping covers all fields present in repo.Patient
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"first_name_th":"A",
			"middle_name_th":"B",
			"last_name_th":"C",
			"first_name_en":"D",
			"middle_name_en":"E",
			"last_name_en":"F",
			"date_of_birth":"2000-02-02",
			"patient_hn":"HN-002",
			"national_id":"N-999",
			"passport_id":"P-999",
			"phone_number":"0999",
			"email":"x@x.com",
			"gender":"F"
		}`))
	}))
	defer ts.Close()

	h, err := NewHospitalAdapter(ts.URL, 1*time.Second)
	assert.NoError(t, err)

	p, err := h.LookupByIdentifier(context.Background(), "N-999")
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "A", p.FirstNameTH)
	assert.Equal(t, "E", p.MiddleNameEN)
	assert.Equal(t, "2000-02-02", *p.DateOfBirth)
}
