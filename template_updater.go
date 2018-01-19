// test project main.go
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// Конвертер для перевода текста в cp1251 и utf8, на вход получает байтовую строку
func Convert(bytestr []byte, route string) []byte {
	newbs := make([]byte, len(bytestr)*2)
	// смена кодировки на ut8
	if route == "utf8" {
		bs1 := charmap.Windows1251.NewDecoder()
		n, _, err := bs1.Transform(newbs, bytestr, false)
		if err != nil {
			panic(err)
		}
		newbs = newbs[:n]
	}
	//смена кодировки на cp1251
	if route == "cp1251" {

		bs1 := charmap.Windows1251.NewEncoder()
		n, _, err := bs1.Transform(newbs, bytestr, false)
		if err != nil {
			panic(err)
		}
		newbs = newbs[:n]
	}

	return newbs
}

// Замена версии в файле ini
func swapVersion(in_file string, out_file string, conf string, new_version string) {
	bs, err := ioutil.ReadFile(in_file)
	if err != nil {
		return
	}

	str := string(Convert(bs, "utf8"))
	lines := strings.Split(str, "\n")

	var version string
	for i, line := range lines {
		if strings.Contains(line, conf) {
			version = lines[i+4][4 : len(lines[i+4])-1]
		}
	}

	newstr := strings.Replace(str, version, new_version, 3)

	ioutil.WriteFile(out_file, Convert([]byte(newstr), "cp1251"), 0777)
}

// создаем строки конфигураций для базы
func conf_generator(conf Configuration, i int) string {
	var mft_config string
	if i == 1 {
		mft_config = "[Config1]\n"
		mft_config += "Catalog=" + conf.catalog + "\n"
		mft_config += "Destination=1C\\" + conf.name_en + "\n"
		mft_config += "Source=1Cv8.cf\n"
	} else if i == 2 {
		mft_config = "[Config2]" + "\n"
		mft_config += "Catalog=" + conf.catalog + " (демо)\n"
		mft_config += "Destination=1C\\" + conf.name_en_demo + "\n"
		mft_config += "Source=1Cv8.dt\n"
	} else if i == 3 {
		mft_config = "[Config3]" + "\n"
		mft_config += "Catalog=" + conf.catalog + " (демо, государственное учреждение)\n"
		mft_config += "Destination=1C\\DemoBudg" + conf.name_en + "\n"
		mft_config += "Source=GOS.dt\n"
	}
	return mft_config

}

// создаем файл mft
func mft_generation(conf Configuration, new_version string, mft_file string) {
	mft := "Vendor=Фирма \"1C\"\n"
	mft += "Name=" + conf.name_ru + "\n"
	mft += "Version=" + new_version + "\n"
	mft += "AppVersion=8.3\n"
	mft += conf_generator(conf, 1)
	mft += conf_generator(conf, 2)
	if conf.gos == "1" {
		mft += conf_generator(conf, 3)
	}
	ioutil.WriteFile(mft_file, []byte(mft), 0777)
}

type Configuration struct {
	id           string
	id_demo      string
	name_ru      string
	gos          string
	catalog      string
	name_en      string
	name_en_demo string
	code         string
}

// Читаем csv файл в структуру
func ReadCsv(csv_path string) []Configuration {
	var config []Configuration
	csvFile, _ := os.Open(csv_path)
	defer csvFile.Close()
	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		config = append(config, Configuration{
			id:           line[0],
			id_demo:      line[1],
			name_ru:      line[2],
			gos:          line[3],
			catalog:      line[4],
			name_en:      line[5],
			name_en_demo: line[6],
			code:         line[7],
		})
	}
	return config
}

// Создаем папку в папке шаблонов и закачиваем туда наши файлики
func copy(source string, destination string, new_version string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	dst := destination + "\\" + strings.Replace(new_version, ".", "_", 4)
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	files, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}
	for _, info := range files {
		if err := fcopy(
			filepath.Join(source, info.Name()),
			filepath.Join(dst, info.Name()),
			info,
		); err != nil {
			return err
		}
	}
	return nil
}

func fcopy(src string, dst string, info os.FileInfo) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return nil
}

func main() {
	csv_path := flag.String("csv", "\\\\dc\\e$\\templates\\base.csv", "Path to csv file( Default:\\\\dc\\e$\\templates\\base.csv")
	id := flag.String("id", "NO", "Conf ID, Example: UT or ACCKZ30(from list.ini)")
	in_file := flag.String("list", "\\\\dc\\e$\\templates\\list.ini", "Path to list.ini file")
	out_file := flag.String("listout", *in_file, "Path to destination list.ini(IF you need test")
	new_version := flag.String("v", "NO", "new version of configuration")
	source := flag.String("d", "NO", "template directory")
	mft_file := *source + "\\1Cv8.mft"
	if *id == "NO" {
		fmt.Println("ID required")
		return
	}
	if *new_version == "NO" {
		fmt.Println("version required")
		return
	}
	if *source == "NO" {
		fmt.Println("template directory required")
		return
	}
	conf := ReadCsv(*csv_path)
	var g int
	for i, c := range conf {
		if c.code == *id {
			g = i
			break
		}
	}
	dest := flag.String("dest", "\\\\sql-backup\\r$\\share\\1C\\82\\Шаблоны\\", "Destination path for template")
	destination := *dest + conf[g].name_en
	mft_generation(conf[g], *new_version, mft_file)
	conf_main := "[Conf" + conf[g].id + "]"
	if conf[g].gos == "1" {
		id_n, err := strconv.Atoi(conf[g].id_demo)
		if err != nil {
			return
		}
		id_n = id_n - 1
		id_doc := strconv.Itoa(id_n)
		conf_doc := "[Conf" + id_doc + "]"
		swapVersion(*in_file, *out_file, conf_doc, *new_version)
	}
	conf_demo := "[Conf" + conf[g].id_demo + "]"
	swapVersion(*in_file, *out_file, conf_main, *new_version)
	swapVersion(*in_file, *out_file, conf_demo, *new_version)
	err := copy(*source, destination, *new_version)
	if err != nil {
		return
	}
}
