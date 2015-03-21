package rawdata

import (
	"errors"
	"fmt"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	structs "shared/structs"
	strconv "strconv"
)

// TODO: engineer this whole mess better.

/**
 * Fetch a set of raw_games that include information related to the specified
 * game. Note that each GameResponse will only include data for a single player
 * so it's better to check processed_game logs if you're looking for more complete
 * game views.
 */
func GetPartialGames(gameid int) []structs.GameResponse {
	gr := make([]structs.GameResponse, 0)
	session, cerr := mgo.Dial("localhost")
	if cerr != nil {
		fmt.Println("Cannot connect to mongodb instance")
		return make([]structs.GameResponse, 0)
	}
	defer session.Close()

	collection := session.DB("league").C("raw_games")
	iter := collection.Find(bson.M{"response.games.gameid": gameid}).Iter()

	result := structs.GameResponseWrapper{}

	for iter.Next(&result) {
		gr = append(gr, result.Response)
	}

	return gr
}

/**
 * Get a list of all complete games (games with stats available) for the provided
 * summoner ID.
 */
func GetCompleteGamesBySummoner(summoner_id int) []structs.GameResponse {
	gr := make([]structs.GameResponse, 0)
	session, cerr := mgo.Dial("localhost")
	if cerr != nil {
		fmt.Println("Cannot connect to mongodb instance")
		return make([]structs.GameResponse, 0)
	}
	defer session.Close()

	collection := session.DB("league").C("raw_games")
	iter := collection.Find(bson.M{"response.summonerid": summoner_id}).Iter()

	result := structs.GameResponseWrapper{}

	for iter.Next(&result) {
		gr = append(gr, result.Response)
	}

	return gr
}

func GetIncompleteGameIdsBySummoner(summoner_id int) []int {
	gameIds := make([]int, 0)
	session, cerr := mgo.Dial("localhost")
	if cerr != nil {
		fmt.Println("Cannot connect to mongodb instance")
		return gameIds
	}
	defer session.Close()

	collection := session.DB("league").C("raw_games")
	iter := collection.Find(bson.M{
		"response.games.fellowplayers.summonerid": summoner_id,
	}).Iter()
	// TODO: Add these back.
	// , bson.M{
	// 	"response.games.gameid":                   true,
	// 	"response.games.fellowplayers.summonerid": true,
	// }).Iter()

	result := structs.GameResponseWrapper{}

	// Iterate through all results and find game ID's that contain this player as a
	// fellow player.
	for iter.Next(&result) {
		for _, game := range result.Response.Games {
			for _, player := range game.FellowPlayers {
				if player.SummonerId == summoner_id {
					gameIds = append(gameIds, game.GameId)
				}
			}
		}
	}

	return gameIds
}

/**
 * Returns the most recent raw_league result for the provided summoner.
 */
func GetLatestLeague(summoner_id int, queue_type string) (structs.LeagueResponseTier, error) {
	// Connect to Mongo and search for entries for this summoner.
	session, cerr := mgo.Dial("localhost")
	if cerr != nil {
		fmt.Println("Cannot connect to mongodb instance")
		return structs.LeagueResponseTier{}, cerr
	}
	defer session.Close()

	collection := session.DB("league").C("raw_leagues")
	iter := collection.Find(bson.M{
		"response." + strconv.Itoa(summoner_id): bson.M{"$exists": true},
	}).Iter()

	// Check to make sure that at least one result came back. If so, iterate through all results to
	// find the most recent one. If not, return an error.
	result := structs.LeagueResponseWrapper{}
	success := iter.Next(&result)
	if !success {
		return structs.LeagueResponseTier{}, errors.New("No matches found for summoner " + strconv.Itoa(summoner_id))
	}

	latest := result
	for iter.Next(&result) {
		if result.Metadata.RequestTime > latest.Metadata.RequestTime {
			latest = result
		}
	}

	for _, tier := range latest.Response[strconv.Itoa(summoner_id)] {
		if tier.Queue == queue_type {
			return tier, nil
		}
	}

	return structs.LeagueResponseTier{}, errors.New("No matches found for queue type " + queue_type)
}
