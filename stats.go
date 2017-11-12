package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
)

type runtimeT struct {
	Mean       float64
	Stddev     float64
	Vari       float64
	RuntimeSum float64
	Runs       int
}

func computeRuntimeStats(runtime []time.Duration) runtimeT {
	var stat runtimeT
	var runtimeSeconds []float64
	for _, r := range runtime {
		runtimeSeconds = append(runtimeSeconds, r.Seconds())
	}

	// TODO handle error?
	stat.Mean, _ = stats.Mean(runtimeSeconds)
	stat.Stddev, _ = stats.StandardDeviation(runtimeSeconds)
	stat.Vari, _ = stats.Variance(runtimeSeconds)
	stat.RuntimeSum, _ = stats.Sum(runtimeSeconds)

	stat.Runs = len(runtime)

	return stat
}

func openStatsFile() (*os.File, error) {
	var statsFile *os.File
	if _, err := os.Stat("stats"); os.IsNotExist(err) {
		// stats does not exist
		statsFile, err = os.Create("stats")
		if err != nil {
			return nil, fmt.Errorf("Error while creating file: %v", err)
		}

		// write header
		statsFile.WriteString("cmd \t avg. runtime (s) \t std. dev. \t variance \t runs \t CAT \t co-slowdown\n")
	} else {
		statsFile, err = os.OpenFile("stats", os.O_WRONLY|os.O_APPEND, 0777)
		if err != nil {
			return nil, fmt.Errorf("Error while opening file: %v", err)
		}
	}
	return statsFile, nil
}

func printStats(c string, stat runtimeT, catMask uint64) {
	s := fmt.Sprintf("%v \t %9.2fs avg. runtime \t %1.6f std. dev. \t %1.6f variance \t %3d runs", c, stat.Mean, stat.Stddev, stat.Vari, stat.Runs)
	if *cat {
		s += fmt.Sprintf("\t %6x CAT", catMask)
	} else {
		s += "\t           "
	}

	ref, ok := referenceRuntimes[c]
	if ok {
		s += fmt.Sprintf("\t %1.6f co-slowdown", stat.Mean/ref.Mean)
	} else {
		s += "\t ref missing"
	}

	fmt.Println(s)
}

func writeToStatsFile(statsFile *os.File, c string, stat runtimeT, catMask uint64) error {
	s := fmt.Sprintf("%v \t %v \t %v \t %v \t %v", c, stat.Mean, stat.Stddev, stat.Vari, stat.Runs)
	if *cat {
		s += fmt.Sprintf("\t %6x", catMask)
	} else {
		s += "\t       "
	}

	ref := referenceRuntimes[c]
	s += fmt.Sprintf("\t %1.6f", stat.Mean/ref.Mean)

	s += "\n"

	_, err := statsFile.WriteString(s)
	if err != nil {
		return err
	}

	return nil
}

func processRuntime(id int, cPair [2]string, catMasks [2]uint64, runtimes [][]time.Duration) error {

	statsFile, err := openStatsFile()
	if err != nil {
		return err
	}
	defer statsFile.Close()

	for i, runtime := range runtimes {
		stat := computeRuntimeStats(runtime)

		printStats(cPair[i], stat, catMasks[i])
		writeToStatsFile(statsFile, cPair[i], stat, catMasks[i])
	}

	fmt.Print("\n")

	for i, runtime := range runtimes {

		filename := fmt.Sprintf("%v-%v", id, i)
		if *cat {
			filename += fmt.Sprintf("-%x", catMasks[i])
		}
		filename += ".time"

		header := "# runtime in nanoseconds of \"" + cPair[i] + "\" on CPUs " + cpus[i] + " while \"" + cPair[(i+1)%2] + "\" was running on cores " + cpus[(i+1)%len(cpus)]
		if *cat {
			header += fmt.Sprintf(" with CAT %6x ", catMasks[i])
		}
		header += "\n"

		file, err := createStatsFile(filename, header)
		if err != nil {
			return err
		}
		defer file.Close()

		err = writeRuntimeFile(file, runtime)
		if err != nil {
			return err
		}
	}

	return nil
}

func createStatsFile(filename string, header string) (*os.File, error) {
	measurementsFile, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("Error while creating file: %v", err)
	}
	_, err = measurementsFile.WriteString(header)

	return measurementsFile, err
}

func writeRuntimeFile(file *os.File, runtime []time.Duration) error {
	var out string
	for _, r := range runtime {
		out += strconv.FormatInt(r.Nanoseconds(), 10)
		out += "\n"
	}

	_, err := file.WriteString(out)
	if err != nil {
		return fmt.Errorf("Error while writing measurements file: %v", err)
	}
	return nil
}
