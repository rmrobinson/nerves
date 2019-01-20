package gtfs

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
	"go.uber.org/zap"
)

var (
	// ErrUnknownFileName is returned if an unknown file is encountered during parsing.
	ErrUnknownFileName = errors.New("unknown file name encountered")
)

// Dataset represents all the data available in a GTFS-exposed dataset.
type Dataset struct {
	Agencies     []*Agency
	Stops        []*Stop
	Routes       []*Route
	Trips        []*Trip
	StopTimes    []*StopTime
	Calendar     []*Calendar
	CalendarDate []*CalendarDate

	logger *zap.Logger

	path string
}

// NewDataset creates a new dataset structure.
func NewDataset(logger *zap.Logger) *Dataset {
	gocsv.SetCSVReader(gtfsCSVReader)
	return &Dataset{
		logger: logger,
	}
}

// LoadFromFSPath loads the contents of the specified path into this dataset.
func (ds *Dataset) LoadFromFSPath(ctx context.Context, path string) error {
	dirEntries, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	ds.path = path

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		err = ds.parseCSVFile(dirEntry)
		if err != nil {
			ds.logger.Debug("error parsing csv file",
				zap.String("file_name", dirEntry.Name()),
				zap.Error(err),
			)

			if err != ErrUnknownFileName {
				return err
			}
		}
	}
	return nil
}

// LoadFromURL loads the contents of the specified URL into this dataset.
func (ds *Dataset) LoadFromURL(ctx context.Context, url string) error {
	body, err := getPath(ctx, url)
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		err = ds.parseZippedCSVFile(zipFile)
		if err != nil {
			ds.logger.Debug("error parsing zipped csv file",
				zap.String("file_name", zipFile.Name),
				zap.Error(err),
			)

			if err != ErrUnknownFileName {
				return err
			}
		}
	}
	return nil
}

func getPath(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return ioutil.ReadAll(resp.Body)
}

func (ds *Dataset) parseCSVFile(dirent os.FileInfo) error {
	f, err := os.Open(filepath.Join(ds.path, dirent.Name()))
	if err != nil {
		return err
	}
	defer f.Close()

	return ds.parseFile(dirent.Name(), f)
}

func (ds *Dataset) parseZippedCSVFile(zf *zip.File) error {
	f, err := zf.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	return ds.parseFile(zf.Name, f)
}

func (ds *Dataset) parseFile(name string, contents io.Reader) error {
	switch name {
	case "agency.txt":
		err := gocsv.Unmarshal(contents, &ds.Agencies)
		if err != nil {
			return err
		}

	case "routes.txt":
		err := gocsv.Unmarshal(contents, &ds.Routes)
		if err != nil {
			return err
		}

	case "trips.txt":
		err := gocsv.Unmarshal(contents, &ds.Trips)
		if err != nil {
			return err
		}

	case "stops.txt":
		err := gocsv.Unmarshal(contents, &ds.Stops)
		if err != nil {
			return err
		}

	case "stop_times.txt":
		err := gocsv.Unmarshal(contents, &ds.StopTimes)
		if err != nil {
			return err
		}

	case "calendar.txt":
		err := gocsv.Unmarshal(contents, &ds.Calendar)
		if err != nil {
			return err
		}

	case "calendar_dates.txt":
		err := gocsv.Unmarshal(contents, &ds.CalendarDate)
		if err != nil {
			return err
		}

	default:
		return ErrUnknownFileName
	}

	return nil
}

// This allows us to handle the fact that GTFS supports optional fields
// We do not error if the CSV row has fewer columns than the header row, for better or worse.
func gtfsCSVReader(in io.Reader) gocsv.CSVReader {
	csvReader := csv.NewReader(in)
	csvReader.FieldsPerRecord = -1
	return csvReader
}
