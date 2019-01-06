package main

import (
	"fmt"
	html2 "html"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bmaupin/go-epub"
)

func main() {

	numberOfProvidedArgs := len(os.Args)

	if numberOfProvidedArgs != 5 {
		log.Fatal("Four arguments required\nUsage: vsi <inputFolder> <outputFilename> <title> <author>")
	}

	inputFolder := os.Args[1]
	outputFilename := os.Args[2]
	title := os.Args[3]
	author := os.Args[4]

	e := epub.NewEpub(title + ": A Very Short Introduction")
	e.SetAuthor(author)

	allChapters, err := filepath.Glob(inputFolder + "/*.html")

	if err != nil {
		log.Fatal("Could not find HTML files in directory")
	}

	for _, chapter := range allChapters {
		fmt.Printf("Formatting %v\n", chapter)
		addChapter(e, chapter)

	}

	err = e.Write(outputFilename)

	if err != nil {
		log.Fatal("Could not write file")
	}
}

func addChapter(e *epub.Epub, chapterFileName string) {
	fileMap := make(map[string]string)

	chapterFilesPath := strings.Replace(chapterFileName, ".html", "_files/", -1)

	fmt.Println(chapterFileName)
	fmt.Println(chapterFilesPath)

	allCss, err := filepath.Glob(chapterFilesPath + "*.css")

	if err != nil {
		log.Fatal("Could not read CSS files")
	}

	var joinedCss string
	for _, css := range allCss {
		cssData, err := ioutil.ReadFile(css)

		if err != nil {
			fmt.Printf("Could not read file: %v\n", css)
		}

		joinedCss += string(cssData)
	}

	epubCSSPath, err := e.AddCSS(joinedCss, "")

	file, err := os.Open(chapterFileName)

	if err != nil {
		log.Fatal("Could not open file")
	}

	doc, err := goquery.NewDocumentFromReader(file)

	if err != nil {
		log.Fatal("Could not open document with goquery")
	}

	var chapterTitle string
	doc.Find(".chapTitle").Each(func(i int, s *goquery.Selection) {
		// Clear page number and chapter number
		chapterTitle = regexp.MustCompile(`p. \d*. (.*)`).ReplaceAllString(s.Text(), "$1")

		// Clear just chapter number
		chapterTitle = regexp.MustCompile(`p. \d+([a-zA-Z]+)`).ReplaceAllString(chapterTitle, "$1")
	})

	doc.Find(".chunkBody").Each(func(i int, s *goquery.Selection) {
		s.PrependHtml("<h1>" + chapterTitle + "</h1>")
		html, err := s.Html()

		if err != nil {
			log.Fatal("Failed to read HTML of chapter")
		}

		s.Find("img").Each(func(i int, s *goquery.Selection) {
			imageName, exists := s.Attr("src")

			unescapedImageFileName := filepath.Dir(chapterFileName) + "/" + strings.Replace(html2.UnescapeString(imageName), "%20", " ", -1)
			if !exists {
				return
			}

			imageEpubFilename, err := e.AddImage(unescapedImageFileName, "")

			if err != nil {
				log.Fatal(fmt.Sprintf("Could not add image %v: %v", unescapedImageFileName, err))
			}

			fileMap[imageName] = imageEpubFilename
		})

		for originalSrc, newSrc := range fileMap {
			html = strings.Replace(html, originalSrc, newSrc, -1)
		}

		deleteNavRegex := regexp.MustCompile(`<ul class="div1-nav">.*<\/ul>`)
		deletePageRefRegex := regexp.MustCompile(`<span id="\w*" class="printPage">p\. \d*<\/span>`)
		deletePageRefIconRegex := regexp.MustCompile(`<span title="\w*" class="printPageMark">↵<\/span>`)

		html = deleteNavRegex.ReplaceAllString(html, "")
		html = deletePageRefRegex.ReplaceAllString(html, "")
		html = deletePageRefIconRegex.ReplaceAllString(html, "")

		chapterEpubFilename := strings.Replace(chapterTitle, "?", "", -1) + ".html"

		e.AddSection(html, chapterTitle, chapterEpubFilename, epubCSSPath)
	})

}
