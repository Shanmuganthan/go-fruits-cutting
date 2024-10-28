package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var maxAllowedUser = 20 // Reduced for testing

type Player struct {
	Name         string
	TotalCount   int
	CuttedFruits []string
}

type FruitAction struct {
	Fruit    Fruit
	PlayerId string
}

type Fruit struct {
	Fruit     string
	StartTime time.Time
	ID        int
}

type FruitActionWithIndex struct {
	Fruit string
	index int
}

var (
	userLock                 sync.RWMutex
	queueLock                sync.Mutex
	users                    = make(map[string]Player)
	userJoinedChannel        = make(chan Player, 10)
	userExitedChannel        = make(chan Player, 10)
	cuttingAction            = make(chan FruitAction, 10)
	removeUnCutFruitsChannel = make(chan FruitActionWithIndex, 10)
	queue                    []Fruit
	maxFruitsOnTheScreen     = 5
	wg                       sync.WaitGroup
)

var planets = []string{"Mercury", "Venus", "Earth", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune"}
var fruits = []string{"🍎", "🍊", "🍌", "🍇", "🍈", "🍉", "🥑", "🍓", "🍒", "🍍", "🥧"}

func addUser() {
	id := rand.Intn(maxAllowedUser)
	playerPrefix := rand.Intn(len(planets))
	playerID := fmt.Sprintf("%s_%d", planets[playerPrefix], id)

	userLock.Lock()
	defer userLock.Unlock()

	if _, exists := users[playerID]; !exists {
		log.Printf("🎉 Player ID Generated: %s 🆔", playerID)

		player := Player{Name: playerID}
		userJoinedChannel <- player

	} else {
		log.Printf("🔄 User Exists: Player ID %s already exists. Retrying...", playerID)
		//addUser()
	}
}

func userLeaves(playerID string) {
	userLock.Lock()
	defer userLock.Unlock()

	if player, exists := users[playerID]; exists {
		delete(users, playerID)
		userExitedChannel <- player

	}
}

func addUserMock() {
	log.Println("🌟 New User: Initiating user addition mock...")
	userLock.RLock()
	size := len(users)
	userLock.RUnlock()
	if size < maxAllowedUser {
		for i := 0; i < 5; i++ {
			go addUser()
			time.Sleep(2 * time.Second)
		}
	} else {
		log.Println("⚠️ No available slots to join at the moment.")
	}
}

func dropFruits() {
	queueLock.Lock()
	defer queueLock.Unlock()

	difference := maxFruitsOnTheScreen - len(queue)
	log.Printf("🍏 Available Slots: %d out of %d slots free for fruits", difference, maxFruitsOnTheScreen)
	if difference == 0 {
		log.Printf("🚫 No Slot Available: Current fruits on screen (%d/%d)", len(queue), maxFruitsOnTheScreen)
	} else {
		for i := 0; i < difference; i++ {
			randomNumber := rand.Intn(len(fruits))
			uniqueId := rand.Intn(1000)
			log.Printf("🌠 Dropping Fruit: %s (ID: %d) at %s", fruits[randomNumber], uniqueId, time.Now().Format("15:04:05"))
			queue = append(queue, Fruit{Fruit: fruits[randomNumber], StartTime: time.Now(), ID: uniqueId})
			time.Sleep(1 * time.Second)
		}
	}
}

func RemoveExpiredFruits() {
	queueLock.Lock()
	defer queueLock.Unlock()

	now := time.Now()
	newQueue := make([]Fruit, 0, len(queue))
	for index, fruit := range queue {
		difference := now.Sub(fruit.StartTime)
		if difference.Seconds() > float64(maxFruitsOnTheScreen) {
			removeUnCutFruitsChannel <- FruitActionWithIndex{index: index, Fruit: fruit.Fruit}
		} else {
			newQueue = append(newQueue, fruit)
		}
	}
	queue = newQueue
}

func cutFruit(user Player, selectedFruit int) {
	log.Printf("🔪 Player %s is cutting a fruit!", user.Name)

	queueLock.Lock()
	defer queueLock.Unlock()

	userLock.Lock()
	defer userLock.Unlock()

	newQueue := make([]Fruit, 0, len(queue))
	for index, fruit := range queue {
		if index == selectedFruit {
			log.Printf("🎉 Hurray!! Player %s cut the fruit %s (ID: %d)", user.Name, queue[selectedFruit].Fruit, fruit.ID)
			playerStatus := users[user.Name]
			playerStatus.CuttedFruits = append(playerStatus.CuttedFruits, fruit.Fruit)
			users[user.Name] = playerStatus

		} else {
			newQueue = append(newQueue, fruit)
		}
	}
	queue = newQueue
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("🍇🍉 Fruit Cutting App Started! 🎮")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	removeExpiredFruits := time.NewTicker(3 * time.Second)
	defer removeExpiredFruits.Stop()
	tickerForDrop := time.NewTicker(5 * time.Second)
	defer tickerForDrop.Stop()
	tickerForCutting := time.NewTicker(1 * time.Second)
	defer tickerForCutting.Stop()
	addUserTicker := time.NewTicker(2 * time.Second) // Add a new user every 2 seconds
	defer addUserTicker.Stop()
	go func() {
		for {
			select {
			case data := <-userJoinedChannel:
				userLock.Lock()
				users[data.Name] = data
				userLock.Unlock()
				log.Printf("🚀 New Player Joined: %s 🎉", data.Name)

			case data := <-userExitedChannel:
				log.Printf("👋 Player Left: %s | Total Fruits Cut: %d | Fruits: %v", data.Name, len(data.CuttedFruits), data.CuttedFruits)

			case data := <-removeUnCutFruitsChannel:
				log.Printf("⏰ Timeout! Removing uncutted fruit: %s from the queue.", data.Fruit)

			case <-removeExpiredFruits.C:
				RemoveExpiredFruits()

			case <-tickerForDrop.C:
				dropFruits()

			case <-tickerForCutting.C:
				userLock.RLock()
				if len(users) > 0 {
					log.Println("🎯 Attempting to cut a fruit for a random player...")
					playerId := 1
					if len(users) > 1 {
						playerId = rand.Intn(len(users))
					}
					var user Player
					for _, v := range users {
						if playerId == 0 {
							user = v
							break
						}
						playerId--
					}
					userLock.RUnlock()
					if len(queue) > 0 {
						go cutFruit(user, rand.Intn(len(queue)))
					}
				} else {
					log.Println("🚫 No users available for fruit cutting at the moment.")
					userLock.RUnlock()
				}

			case <-ticker.C:
				log.Println("👻 Mocking: Simulating player leave manually...")
				userLock.RLock()
				if len(users) > 1 {
					for playerID := range users {
						userLock.RUnlock()
						userLeaves(playerID)
						break
					}
				} else {
					userLock.RUnlock()
				}
			case <-addUserTicker.C: // Call addUserMock at regular intervals
				go addUserMock()
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	log.Println("All users have attempted to join.")
	http.ListenAndServe(":8080", nil)
}
