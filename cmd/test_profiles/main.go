package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/embedtools/instagram-scraper/scraper"
	"github.com/embedtools/instagram-scraper/types"
)

var profiles = []string{
	"closedbyansley", "rickycallahan", "aliceyangteam", "homes.with.mariyah",
	"privatemoneyspecialists", "mariosimauchi", "realtorgamechanger", "realestatenetworkca",
	"homeloanswiththeateam", "melissaburgerealtor", "coffee_contracts_cocktails", "curated.realestate.agency",
	"agenthuntersumner", "sellinsoflo", "living_in_central_texas", "realtiiz",
	"miami_realtor_luisana", "theresilientrealtor", "jenniferrodgershomes", "thingsmarleydoes",
	"lizbrownrealtordaily", "aprilbaker_realtor", "realtorsuzanna", "haiyingre",
	"juneaurealtorrachel", "yourstjohnscounty", "justbelieve_7", "closewithcanterbury",
	"tirsoabarcatuagente", "pennyweathers.realtor", "maven_realty_llc", "jermaine_realestate",
	"ambersmith.gprealtor", "ambersellsfl", "sheerasellsswfl", "paradisepalmsrealtygroup",
	"the_eldergroup", "janajagerrealestate", "realtorshobhabaral", "immickeyg",
	"thesteveturner", "thecoreyjones", "akbar_zareh", "floridarealtorshari",
	"soniathesavvyrealtor", "jpanegreiros", "sweethomeauburnal", "damybello.tucasard",
	"floridarealtor_exprealty", "blakekircherofficial", "ztrans", "creativebrandingstudiousa",
	"healthyhomeswithsasha", "realjohnnynoe", "praneeta_chand_realtor", "goldbergtherealtor",
	"laurafloodcoaching", "boldagent", "rapportfurniture", "burchellhometeam",
	"reillytherealtor", "mattledford", "graciela.iglesias.realtor", "the.evanich.group",
	"janet_dale_himmelheber", "jasminempatel", "emencia", "direct_mortgage",
	"rebeccahightowerrealtor", "nwarirealtygroup", "trendyblindsandclosets", "yqrrealtor",
	"mortgages_by_madrid", "mariobravorealtor", "shayesellssantacruz", "tomaballo",
	"kvsindiarealtors", "richardouimet", "ironwillmathews", "terrybarnett",
	"mtpleasantlnl", "plizplaz.social", "richard_ong_batam", "realtyrecordsmedia",
	"crystalbaxleyhomes", "udayarealty_", "chris_smth", "jaypittsrealtor",
	"gloverucoaching", "hannah_thepropertygeek", "levilascsak", "bevhumanrealtor",
	"haleyswankrealtor", "painteddoorrealty", "schmidt_zak", "movemetodfw",
	"hersy.realestate", "eva_bogotlieva", "dblacktie", "jason_stanley_realtor",
	"tampabaydreamhomes", "dfw.realtor", "nelviabullockrealtor", "mariesuchyhomes",
	"_heathwagner_", "staceysellsennistx", "myagentsarahhjort", "theshelbyshow__",
	"jenctx", "edh_realtor", "agentmannix", "teresaryanandco",
	"tiandraelliott", "iszac_rose", "davedesilva", "agenttenillecarlosbey",
	"ninepgh_realtor", "raquelferreira.remax", "deeannhotterealtor", "nhoramontereycounty",
	"crystalwattrealtor", "atnasko", "chrissellsct", "autumndeason.realtor",
	"realtorkeena", "kimfraserrealtor", "nexthomewithcasey", "lifeinmoco",
	"mindcoachsteph", "jeffwallace24", "thepeakyrealtor", "brokerchristina",
	"amkenglobalpropertiesltd", "noramyagent", "thereallaurasikorski", "jennyoleyva",
	"kevinirwin_", "manuelescotet", "dianavrealestate", "itsnikkigil",
	"annalisa.sells.houses", "tanishatherealtor", "kyle.lounsbury", "jentolleyatlanta",
	"rahelchoi.realtor", "chadratto", "derealestateexpert", "charisse_sells_flhomes",
	"itskarenitteilag", "brownaddition", "tomsellsmesquite", "andrew.itzsold",
	"austinstjean", "benwheelerrealtor", "loansbykerry", "rworldypn",
	"jennyo_realtor", "allisonluxuryhomesgroup", "jesseandsamantha", "jackicampbellhomes",
	"jens.h.nielsen", "dustinsiklerrealestate", "soldbuyteam_realtors", "faybrink_houstonrealtor",
	"jeffirealtor", "credimaxusa", "ihomescabo", "marianbarilerealtor",
	"desa_denis", "jpereirayouragent", "marianbarilerealtor", "theexperiencenjteam",
	"_heathwagner_", "elite_digital1", "legacytxrealty", "homesbyjlo",
	"sparklesus721", "joyscovell.realtor", "jessicataylorrealty", "anniesells.la",
	"katecaffrey", "lanasrealny", "erika_rodan.loops.keytime", "denise_is_swift",
	"khalidnathanaleem", "liferealtyandmoorenc", "benpollard6", "joegrantrealtor",
	"taraoatesrealtor", "bayarea.realestatebyseema", "emily_nc_realestate", "janay.taua.realtor",
	"amandageller_", "inatayhriscay_realtay", "keyswithchristine", "itsdeecain",
	"karinmorabito_realtor", "ali.mohseni.atx", "call.kelly.hall", "homesbycherylbowman",
	"floridalivingwithmelissa", "charita_abesa_realestate", "callummoloneyrealtor", "ashleykjohnson_",
	"phantomassistants", "thenagygroup", "angelasancheztxrealtor", "baezbroker",
	"in_and_around_raleigh", "penney_groupexp", "realtorjay44", "closewithclancy",
	"sellwith_elle", "joshuamarriott", "realtorberni", "realtor_tayleonpuryear",
	"louiscornejo.realtor", "sianasummeyrealestate", "rebecca_duncan_real_estate", "jamielee_eang",
	"soldbycandi", "jenngentilerealtor", "paasch_properties", "the_cockerille_group.realtor",
	"jeaston11", "wegotcobbs", "bestwaproperties", "projectfoundry",
	"lidiyasrealestate", "yasminrogersrealestate", "georgianarunsjanas", "mrnevadahomes",
	"rondawhiterealtor", "toyinsellshouses", "tessierpropertygroup", "brandont.realty",
	"alexander_thegreat_polanco", "thewaltzmanteam", "mortgageswithpaul", "amydsellsnc",
	"3kcleaningservice", "kaykuehn", "wanderlust_creations_ia", "elite__uniquehomesltd",
	"heynowhomes", "zakary_dana_chicago_realtor", "jsellsatl", "rogersells805",
	"mattapoisettlivingblog", "agentpodcast", "j.nstylnnn", "mr.ucnj",
	"joeyestates", "harmonypress718", "burciaga.media", "ideasforrealestate",
	"augustprobuilders", "epiquejosh", "lukeacree", "jenny.celly.realtor",
	"nathalysalas_", "daisylayland_estateagent", "viviendoengeorgia", "kimberlysuazo_realtor",
	"hofferhomes", "erinpiersonmills", "april.landes.realtor", "hoosierheather317",
	"megbradley_realtor", "sellwithmadison", "hrvahomes", "tavaresrajasingam",
	"nora.hinojosa_realtor", "cataleonschmid_realtor", "jmk.estates", "jeffpfitzer",
	"mji8128", "mrs_most_aggressive_realtor", "korynealrealestate", "nbtxliving",
	"vrivera1630", "jenmason_yourrealestateagent", "iamsonalihutson", "kim.carmichael.co",
	"glendawardrealtor", "stevenblevinerealtor", "moreymelissa", "ajparazzi",
	"debbie.welch.realtor", "sam.i.am.houses", "alanterealestate", "thetracyvertus",
	"nickdhomestn", "kristenpetersonrealtor", "jphebertremax", "carlos_and_kristy_in_sma",
	"williamhaganre", "tyevansrealtor", "blueinkhomes_realtors", "moongroupchicago",
	"lthomashtx_realtor", "lindsay.reddy", "alchemyvacationproperties", "savoieamyrealtor",
	"haydenkobertrealestate", "maytepachecorealtor", "elsie_gomez_rehustler", "cliftonjohnsonicon10x",
	"lannomrealestate", "springfieldmissouri.homes", "kaseysingleton_", "melmelklein",
	"ingatlanmarketinges", "madelinesmallwood", "melissawurealty", "brock_bremmer_realtor",
	"frankiefahimi", "dhillyerhomes", "jbvadvisors", "marcrutledge",
	"the.information.guys", "stevenmitchemallentate", "realtortatenda", "prothrogroupsa",
	"patmjohnson09", "bettercallpaulatx", "susannakunkel", "tullysellshomes",
	"bw_at_exp", "noor_realestate.kr", "solarmelby", "meganslaatsrealtor",
	"kimblelendingteam", "asesorespegaso", "damionwagner", "close.withus",
	"sosgraphics_gp", "cmoorehomes", "leasedbydillonross", "homes_by_lina",
	"all_gona_wrong", "annettemartinezrealtor", "deisyhtxrealtor", "joe_sells_texas",
	"tatyana_reynolds_", "hazelrealestatedurango", "yassarayub", "ashleyarthurrealestate",
	"maria.ingle.realestate", "realtormelissatremus", "tamaraosoriorealtor", "imrudycoronel",
	"nicoleelizabeth.realestate", "suzykormanik", "atx_homewithflores", "dianalewisrealtor",
	"katetofurirealestate", "belladonnariso", "sara.your.home.girl", "veronicasellsthevalley",
	"dorsey_sells_dfw", "dianapeters_realtor", "allytrod", "sanfordsurrounded",
	"willmchugh_", "jordyluxetransactions", "franciscojlara", "tonyfloresrealtor",
	"leeroy_kusto", "thebricgroup", "itstinakirkpatrick", "kristiedaniels_realtor",
	"emilyblatt_", "shannecarvalho_re_team", "maijiadonovanrealtor", "tracey_your_texas_realtor",
	"azrealtormarielena", "swift_and_co_realty", "luxehomemanny", "realestatebykaylynn",
	"my.realtorlife", "dawnacappshouses", "homewithkristin", "joserealtorporterranch",
	"christine.m.brosseau", "john.desmedt", "sellingwithcaitlyn", "toniagambill",
	"linocrisostomo.consultor.imob", "janiesealton_realtor", "jb_digitalltd", "theagentaid",
	"cretanrealestate", "ladyboss_sp", "angelrupertrealtor", "balmodov",
	"mickey.cavazos", "craig0219", "angels.agency00", "tracigagnon.realtor",
}

type profileResult struct {
	Username string                 `json:"username"`
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
	Private  bool                   `json:"private,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Elapsed  string                 `json:"elapsed"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

func main() {
	proxyURL := os.Getenv("PROXY_URL")
	curlBin := os.Getenv("CURL_BIN_PATH")

	if proxyURL == "" {
		fmt.Println("ERROR: PROXY_URL env var is required")
		os.Exit(1)
	}

	opts := []scraper.Option{
		scraper.WithProxyURL(proxyURL),
	}
	if curlBin != "" {
		opts = append(opts, scraper.WithCurlBinPath(curlBin))
	}

	client, err := scraper.New(opts...)
	if err != nil {
		fmt.Printf("ERROR creating client: %v\n", err)
		os.Exit(1)
	}

	batchSize := 50
	var (
		allResults []profileResult
		mu         sync.Mutex
		passed     int64
		failed     int64
	)

	totalStart := time.Now()

	for batchStart := 0; batchStart < len(profiles); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(profiles) {
			batchEnd = len(profiles)
		}
		batch := profiles[batchStart:batchEnd]
		batchNum := (batchStart / batchSize) + 1
		totalBatches := (len(profiles) + batchSize - 1) / batchSize

		fmt.Printf("\n=== Batch %d/%d (%d profiles) ===\n", batchNum, totalBatches, len(batch))
		batchStartTime := time.Now()

		var wg sync.WaitGroup
		for _, username := range batch {
			wg.Add(1)
			go func(u string) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				start := time.Now()
				out, err := client.GetProfile(ctx, &types.GetProfileInput{URL: u})
				elapsed := time.Since(start)

				r := profileResult{
					Username: u,
					Elapsed:  elapsed.Round(time.Millisecond).String(),
				}

				if err != nil {
					r.Success = false
					r.Error = err.Error()
					atomic.AddInt64(&failed, 1)
					fmt.Printf("  FAIL  %-40s %s  %s\n", u, elapsed.Round(time.Millisecond), err.Error())
				} else {
					r.Success = true
					r.Private = out.IsPrivate
					r.Data = out.Data
					if name, ok := out.Data["full_name"].(string); ok {
						r.Name = name
					}
					atomic.AddInt64(&passed, 1)
					status := "OK"
					if out.IsPrivate {
						status = "PRIVATE"
					}
					fmt.Printf("  %-7s %-40s %s  %s\n", status, u, elapsed.Round(time.Millisecond), r.Name)
				}

				mu.Lock()
				allResults = append(allResults, r)
				mu.Unlock()
			}(username)
		}
		wg.Wait()

		fmt.Printf("  Batch done in %s\n", time.Since(batchStartTime).Round(time.Millisecond))

		if batchEnd < len(profiles) {
			fmt.Println("  Waiting 5s before next batch...")
			time.Sleep(5 * time.Second)
		}
	}

	totalElapsed := time.Since(totalStart)

	fmt.Printf("\n========================================\n")
	fmt.Printf("TOTAL:   %d profiles in %s\n", len(profiles), totalElapsed.Round(time.Second))
	fmt.Printf("PASSED:  %d\n", passed)
	fmt.Printf("FAILED:  %d\n", failed)
	fmt.Printf("RATE:    %.1f profiles/min\n", float64(len(profiles))/totalElapsed.Minutes())
	fmt.Printf("========================================\n")

	outFile := "profile_test_500_results.json"
	f, err := os.Create(outFile)
	if err == nil {
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"total":   len(profiles),
			"passed":  passed,
			"failed":  failed,
			"elapsed": totalElapsed.String(),
			"results": allResults,
		})
		f.Close()
		fmt.Printf("Full results written to %s\n", outFile)
	}
}
