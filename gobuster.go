/*-----------------------------------------------------------------
Nom du programme : Gobuster
Description : Gobuster peut effectuer un scan sur l'arboresence d'un
			  site, à la recherche de fichiers sensibles, cachés ou
			  vulnérables
Auteur : Aymen HAMDI
Date de création : 06/01/2024
Version : V1
Usage standard : go run main.go -t <site cible> -d <chemin vers un
		fichier de dictionnaire>
-----------------------------------------------------------------*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Result représente le résultat d'un test
type Result struct {
	URL    string
	Status int
}

// Loggers pour la journalisation du scan et les erreurs
var scanLogger, errorLogger *log.Logger

/*
Fonction principale du programme
*/

func main() {

	// Déclaration des flags
	dictionaryPath := flag.String("d", "", "Path to dictionary file")
	quietMode := flag.Bool("q", false, "When set to true, only show HTTP 200")
	showAll := flag.Bool("a", false, "When set to true, show all results, including non-200 status codes")
	target := flag.String("t", "", "Target to enumerate")
	workers := flag.Int("w", 1, "Number of workers to run")
	logToFile := flag.Bool("l", false, "When set to true, log results to a file")

	// Parsing des flags
	flag.Parse()

	errorLogFile, err := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error : Something went wrong when opening error.log file - ", err)
		return
	}
	defer errorLogFile.Close()

	errorLogger = log.New(errorLogFile, "", log.LstdFlags)

	// Vérification du format pour le flag target
	if !strings.HasPrefix(*target, "http://") && !strings.HasPrefix(*target, "https://") {
		fmt.Println("Error : The flag \"target\" must begin with 'http://' ou 'https://'.")
		errorLogger.Println("Error : The flag \"target\" must begin with 'http://' ou 'https://'.")
		return
	}

	// Vérification des flags obligatoires
	if *dictionaryPath == "" || *target == "" {
		fmt.Println("Error : Please specify the path to dictionary file (-d) and the target (-t).")
		errorLogger.Println("Error : Missing dictionnary path or target")
		return
	}

	// Initialisation du logger pour la journalisation
	if *logToFile {

		logFile, err := os.OpenFile("scan.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error : Something went wrong when opening scan.log file - ", err)
			errorLogger.Println("Error : Something went wrong when opening scan.log file - ", err)
			return
		}
		defer logFile.Close()

		// Configuration du logger pour écrire dans le fichier
		scanLogger = log.New(logFile, "", log.LstdFlags)
	}

	// Lecture du dictionnaire
	dictionary, err := readDictionary(*dictionaryPath)
	if err != nil {
		return
	}

	// Affiche le résultat de la vérification de la connectivité
	if err := checkAndSet(target); err != nil {
		return
	}

	// Affichage des paramètres saisis par l'utilisateur
	fmt.Println("---")
	fmt.Println("Target:", *target)
	fmt.Println("List:", *dictionaryPath)
	fmt.Println("Dictionary Size:", len(dictionary))
	fmt.Println("Workers:", *workers)
	fmt.Println("---")

	// Démarrage du scan
	fmt.Println("Starting scan...")
	startTime := time.Now()

	results := make(chan Result)
	var wg sync.WaitGroup

	// Divise le dictionnaire entre les workers
	chunkSize := len(dictionary) / *workers
	chunks := make([][]string, *workers)
	for i := range chunks {
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if i == *workers-1 {
			// Le dernier worker inclut les éléments restants
			end = len(dictionary)
		}
		chunks[i] = dictionary[start:end]
	}

	// Démarrage des workers
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runScanner(*target, chunks[workerID], results)
		}(i)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Affichage des résultats et enregistrement dans le fichier scan.log
	for result := range results {
		if *quietMode {
			if result.Status == http.StatusOK {
				fmt.Println(result.URL)
			}
		} else {
			if result.Status == http.StatusOK || *showAll {
				fmt.Printf("%s %d\n", result.URL, result.Status)
			}
		}

		if *logToFile {
			scanLogger.Printf("%s %d\n", result.URL, result.Status)
		}
	}

	fmt.Printf("Scan done in %fs\n", time.Since(startTime).Seconds())
}

/*
Fonction qui lit un fichier de dictionnaire et renvoie les lignes lues
Variables :

	path (string)
	file (*File)
	err (error)
	lines ([]string)
	scanner (*Scanner)
*/
func readDictionary(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error : Something went wrong when reading the dictionary file - ", err)
		errorLogger.Println("Error : Something went wrong when reading the dictionary file - ", err)
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error : Something went wrong when reading the dictionary file - ", err)
		errorLogger.Println("Error : Something went wrong when reading the dictionary file - ", err)
		return nil, err
	}

	return lines, nil
}

/*
Fonction qui effectue le scan sur la cible avec le dictionnaire choisi
Variables :

	target (string)
	dictionary ([]string)
	results (chan)
	url (string)
	err (error)
	resp (*Response)
*/
func runScanner(target string, dictionary []string, results chan<- Result) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	for _, path := range dictionary {
		url := fmt.Sprintf("%s%s", target, path)
		resp, err := client.Get(url)
		if err != nil {
			// En cas d'erreur, on suppose que le chemin n'existe pas
			results <- Result{URL: url, Status: http.StatusNotFound}
		} else {

			results <- Result{URL: url, Status: resp.StatusCode}
			resp.Body.Close()
		}
	}
}

/*
Fonction qui vérifie au préalable la connectivité vers la cible.
En cas d'échec, teste une deuxième connexion avec HTTP ou HTTPS.
Gère les redirections
Variables :

	protocol (string)
	target (string)
	err (error)
	client (Client)
	resp (*Response)
	errorLogger (*log.Logger)
	scanner (*Scanner)
	answer (string)
	redirectURL (string)
	parsedURL (string)
*/
func checkAndSet(target *string) error {

	var protocol string

	if strings.HasPrefix(*target, "https://") {
		protocol = "HTTPS"
	} else {
		protocol = "HTTP"
	}

	fmt.Printf("Checking connectivity (%s)... ", protocol)

	client := http.Client{
		Timeout: 10 * time.Second,
		// Désactive les redirections
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head(*target)
	if err != nil {
		fmt.Println("Failed")
		errorLogger.Println("Error : Connectivity check failed with protocol : ", protocol, " - ", err)

		// Inverse le protocole et réessaie
		if protocol == "HTTPS" {
			*target = strings.Replace(*target, "https://", "http://", 1)
			protocol = "HTTP"
		} else {
			*target = strings.Replace(*target, "http://", "https://", 1)
			protocol = "HTTPS"
		}

		fmt.Printf("Checking connectivity (%s)... ", protocol)
		resp, err = client.Head(*target)
		if err != nil {
			fmt.Println("Failed")
			fmt.Println("Error : No connection could be established with the host - ", err)
			errorLogger.Println("Error : Connectivity check failed - ", err)
			return err
		}
	}

	defer resp.Body.Close()

	fmt.Println("OK")

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		fmt.Println("Redirection detected")
		fmt.Printf("Original target: %s\n", *target)
		fmt.Printf("Redirected to: %s\n", resp.Header.Get("Location"))

		// Demande à l'utilisateur s'il veut continuer le scan sur la nouvelle cible
		for {
			fmt.Print("Do you want to continue the scan on the new target? (y/n): ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := strings.ToLower(scanner.Text())
			if answer == "y" {
				redirectURL := resp.Header.Get("Location")

				// Extrait le domaine principal de l'URL de redirection
				parsedURL, err := url.Parse(redirectURL)
				if err == nil {
					*target = parsedURL.Scheme + "://" + parsedURL.Host
				} else {
					// En cas d'erreur, utilise l'URL complète
					*target = redirectURL
				}

				checkAndSet(target)
				break
			} else if answer == "n" {
				fmt.Println("Scan aborted by user.")
				os.Exit(0)
			} else {
				fmt.Println("Invalid choice. Please enter 'y' for yes or 'n' for no.")
			}
		}
	}

	return nil
}
