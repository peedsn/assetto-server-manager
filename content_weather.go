package servermanager

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cj123/ini"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const weatherInfoFile = "weather.ini"

type Weather map[string]string

// defaultWeather is loaded if there aren't any weather options on the server
var defaultWeather = Weather{
	"1_heavy_fog":    "Heavy Fog",
	"2_light_fog":    "Light Fog",
	"3_clear":        "Clear",
	"4_mid_clear":    "Mid Clear",
	"5_light_clouds": "Light Clouds",
	"6_mid_clouds":   "Mid Clouds",
	"7_heavy_clouds": "Heavy Clouds",
}

func ListWeather() (Weather, error) {
	baseDir := filepath.Join(ServerInstallPath, "content", "weather")

	weatherFolders, err := ioutil.ReadDir(baseDir)

	if os.IsNotExist(err) {
		return defaultWeather, nil
	} else if err != nil {
		return nil, err
	}

	weather := defaultWeather

	for _, weatherFolder := range weatherFolders {
		if !weatherFolder.IsDir() {
			continue
		}

		// read the weather info file
		name, err := getWeatherName(baseDir, weatherFolder.Name())

		if err != nil {
			return nil, err
		}

		if name == "" {
			name = weatherFolder.Name()
		}

		weather[weatherFolder.Name()] = name
	}

	if len(weather) == 0 {
		return defaultWeather, nil
	}

	return weather, nil
}

func getWeatherName(folder, weather string) (string, error) {
	f, err := os.Open(filepath.Join(folder, weather, weatherInfoFile))

	if err != nil {
		return "", nil
	}

	defer f.Close()

	i, err := ini.Load(f)

	if err != nil {
		return "", err
	}

	s, err := i.GetSection("LAUNCHER")

	if err != nil {
		return "", err
	}

	k, err := s.GetKey("NAME")

	if err != nil {
		return "", nil
	}

	return k.String(), nil
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	weather, err := ListWeather()

	if err != nil {
		logrus.Errorf("could not get weather list, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("content", "weather.html"), map[string]interface{}{
		"weathers": weather,
	})
}

func apiWeatherUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Weather")
}

func weatherDeleteHandler(w http.ResponseWriter, r *http.Request) {
	weatherKey := mux.Vars(r)["key"]
	weatherPath := filepath.Join(ServerInstallPath, "content", "weather")

	existingWeather, err := ListWeather()

	if err != nil {
		logrus.Errorf("could not get weather list, err: %s", err)

		AddFlashQuick(w, r, "couldn't get weather list")

		http.Redirect(w, r, r.Referer(), http.StatusFound)

		return
	}

	var found bool

	for key := range existingWeather {
		if weatherKey == key {
			// Delete car
			found = true

			err := os.RemoveAll(filepath.Join(weatherPath, weatherKey))

			if err != nil {
				found = false
				logrus.Errorf("could not remove weather files, err: %s", err)
			}

			delete(existingWeather, key)

			break
		}
	}

	var message string

	if found {
		// confirm deletion
		message = "Weather preset successfully deleted!"
	} else {
		// inform weather wasn't found
		message = "Sorry, weather preset could not be deleted. Are you sure it was installed?"
	}

	AddFlashQuick(w, r, message)

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}