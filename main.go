package main

import (
	"crypto-exchange/orderbook"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
)

const (
	MarketETH Market = "ETH"

	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"

	exchangePrivateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d" // Ganache 0 id
)

type (
	OrderType string
	Market    string

	PlaceOrderRequest struct {
		UserID int64
		Type   OrderType
		Bid    bool
		Size   float64
		Price  float64
		Market Market
	}

	Order struct {
		UserId    int64
		ID        int64
		Price     float64
		Size      float64
		Bid       bool
		Timestamp int64
	}

	OrderbookData struct {
		TotalBidVolume float64
		TotalAskVolume float64
		Asks           []*Order
		Bids           []*Order
	}

	MatchedOrder struct {
		Price float64
		Size  float64
		ID    int64
	}
)

func main() {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}

	ex, err := NewExchange(exchangePrivateKey, client)
	if err != nil {
		log.Fatal(err)
	}

	pkStr7 := "a453611d9419d0e56f499079478fd72c37b251a94bfde4d19872c44cf65386e3"
	user7 := NewUser(pkStr7, 7)
	ex.Users[user7.ID] = user7

	pkStr8 := "829e924fdf021ba3dbbc4225edfece9aca04b929d6e75613329ca6f1d31c0bb4" // Ganache 8
	user8 := NewUser(pkStr8, 8)
	ex.Users[user8.ID] = user8

	jhonPk := "e485d098507f54e7733a205420dfddbe58db035fa577fc294ebd14db90767a52"
	jhon := NewUser(jhonPk, 666)
	ex.Users[jhon.ID] = jhon

	e.GET("/book/:market", ex.handleGetBook)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)

	e.Start(":3000")
}

type User struct {
	ID         int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(privateKey string, id int64) *User {
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(err)
	}

	return &User{
		ID:         id,
		PrivateKey: pk,
	}
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

type Exchange struct {
	Client     *ethclient.Client
	Users      map[int64]*User
	orders     map[int64]int64
	PrivateKey *ecdsa.PrivateKey
	orderbooks map[Market]*orderbook.Orderbook
}

func NewExchange(privateKey string, client *ethclient.Client) (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketETH] = orderbook.NewOrderBook()

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &Exchange{
		Client:     client,
		Users:      make(map[int64]*User),
		orders:     make(map[int64]int64),
		PrivateKey: pk,
		orderbooks: orderbooks,
	}, nil
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]

	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderbookData{
		TotalBidVolume: ob.BidTotalVolume(),
		TotalAskVolume: ob.AskTotalVolume(),
		Asks:           []*Order{},
		Bids:           []*Order{},
	}

	for _, limit := range ob.Asks() {
		for _, order := range limit.Orders {
			o := Order{
				UserId:    order.UserID,
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Asks = append(orderbookData.Asks, &o)
		}
	}

	for _, limit := range ob.Bids() {
		for _, order := range limit.Orders {
			o := Order{
				UserId:    order.UserID,
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Bids = append(orderbookData.Bids, &o)
		}
	}

	return c.JSON(http.StatusOK, orderbookData)
}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	ob := ex.orderbooks[MarketETH] //todo: remove hardcoded MarketETH
	order := ob.Orders[int64(id)]
	ob.CancelOrder(order)

	return c.JSON(http.StatusOK, map[string]any{"msg": "order deleted"})
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrder) {
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrder, len(matches))

	isBid := false

	if order.Bid {
		isBid = true
	}

	for i := 0; i < len(matchedOrders); i++ {
		id := matches[i].Bid.ID
		if isBid {
			id = matches[i].Ask.ID
		}
		matchedOrders[i] = &MatchedOrder{
			ID:    id,
			Size:  matches[i].SizeFilled,
			Price: matches[i].Price,
		}
	}

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	return nil
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := Market(placeOrderData.Market)
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID)

	if placeOrderData.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{"msg": "limit order placed"})
	}

	if placeOrderData.Type == MarketOrder {
		matches, matchedOrders := ex.handlePlaceMarketOrder(market, order)
		if err := ex.handleMatches(matches); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{"matches": matchedOrders})
	}

	return nil
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Ask.UserID)
		}

		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Bid.UserID)
		}

		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)

		// Only used to exchange fees
		// exchangePubKey := ex.PrivateKey.Public()
		// plubicKeyECDSA, ok := exchangePubKey.(*ecdsa.PublicKey)
		// if !ok {
		// 	return fmt.Errorf("error casting public key to ecdsa")
		// }

		amount := big.NewInt(int64(match.SizeFilled))
		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
	}
	return nil
}
