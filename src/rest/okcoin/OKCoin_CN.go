package okcoin

import
(
	. "rest"
	"strconv"
	"errors"
	"net/http"
	"net/url"
	"encoding/json"
	"strings"
	//"fmt"
)

const
(
	EXCHANGE_NAME = "okcoin_cn";
	url_ticker = "https://www.okcoin.cn/api/v1/ticker.do";
	url_depth = "https://www.okcoin.cn/api/v1/depth.do";
	url_trades = "https://www.okcoin.cn/api/v1/trades.do";
	url_kline = "https://www.okcoin.cn/api/v1/kline.do";

	url_userinfo = "https://www.okcoin.cn/api/v1/userinfo.do";
	url_trade = "https://www.okcoin.cn/api/v1/trade.do";
	url_cancel_order = "https://www.okcoin.cn/api/v1/cancel_order.do";
	url_order_info = "https://www.okcoin.cn/api/v1/order_info.do";	
)

type OKCoinCN_API struct{
	client *http.Client
	api_key string
	secret_key string
}

func currencyPair2String(currency CurrencyPair) string{
	switch currency{
		case BTC_CNY:
			return "btc_cny";
		case LTC_CNY:
			return "ltc_cny";
		default:
			return "";
	}
}

func New(client *http.Client, api_key, secret_key string) * OKCoinCN_API{
	return &OKCoinCN_API{client, api_key, secret_key};
}

func (ctx *OKCoinCN_API) buildPostForm(postForm *url.Values) error {
	postForm.Set("api_key", ctx.api_key);
	//postForm.Set("secret_key", ctx.secret_key);

	payload := postForm.Encode();
	payload = payload + "&secret_key=" +ctx.secret_key;

	sign, err := GetParamMD5Sign(ctx.secret_key, payload);
	if err != nil {
		return err;
	}

	postForm.Set("sign", strings.ToUpper(sign));
	//postForm.Del("secret_key")
	return nil;
}

func (ctx *OKCoinCN_API) placeOrder(side , amount , price string , currency CurrencyPair)(*Order ,error)  {
	postData := url.Values{};
	postData.Set("type" , side);
	postData.Set("amount" , amount);
	postData.Set("price" , price);
	postData.Set("symbol" , currencyPair2String(currency));

	err := ctx.buildPostForm(&postData);

	if err != nil{
		return nil, err;
	}

	body, err := HttpPostForm(ctx.client , url_trade , postData);

	if err != nil{
		return nil, err;
	}

	//println(string(body));

	var respMap map[string]interface{};

	err = json.Unmarshal(body , &respMap);

	if err != nil {
		return nil , err;
	}

	if !respMap["result"].(bool){
		return nil , errors.New(string(body));
	}

	order := new(Order);
	order.OrderID = int(respMap["order_id"].(float64));
	order.Price, _ = strconv.ParseFloat(price, 64);
	order.Amount, _ = strconv.ParseFloat(amount, 64);
	order.Currency = currency;
	order.Status = ORDER_UNFINISH;

	switch side {
	case "buy":
		order.Side = BUY;
	case "sell":
		order.Side = SELL;
	}

	return  order , nil;

}

func (ctx * OKCoinCN_API) LimitBuy(amount, price string, currency CurrencyPair) (*Order, error){
	return ctx.placeOrder("buy" , amount ,price,currency);
}


func (ctx * OKCoinCN_API) LimitSell(amount, price string, currency CurrencyPair) (*Order, error) {
	return ctx.placeOrder("sell" , amount ,price  ,currency);
}

func (ctx * OKCoinCN_API) CancelOrder(orderId string, currency CurrencyPair) (bool, error){
	return false, nil;
}

func (ctx * OKCoinCN_API) GetOneOrder(orderId string, currency CurrencyPair) (*Order, error){
	return nil, nil;
}

func (ctx * OKCoinCN_API) GetUnfinishOrders(currency CurrencyPair) ([]Order, error){
	return nil, nil;
}

func (ctx * OKCoinCN_API) GetAccount() (*Account, error){
	postData := url.Values{};
	err := ctx.buildPostForm(&postData);

	if err != nil{
		return nil, err;
	}

	body, err := HttpPostForm(ctx.client , url_userinfo , postData);

	if err != nil{
		return nil, err;
	}

	var respMap map[string]interface{};

	err = json.Unmarshal(body , &respMap);

	if err != nil {
		return nil , err;
	}

	if !respMap["result"].(bool){
		errcode := strconv.FormatFloat(respMap["error_code"].(float64), 'f', 0, 64);
		return nil, errors.New(errcode);
	}

	info := respMap["info"].(map[string]interface{});
	funds := info["funds"].(map[string]interface{});
	asset := funds["asset"].(map[string]interface{});
	free := funds["free"].(map[string]interface{});
	freezed := funds["freezed"].(map[string]interface{});
	
	account := new(Account);
	account.Exchange = ctx.GetExchangeName();
	account.Asset, _ = strconv.ParseFloat(asset["total"].(string), 64);
	account.NetAsset, _ = strconv.ParseFloat(asset["net"].(string), 64);

	var btcSubAccount SubAccount;
	var ltcSubAccount SubAccount;
	var cnySubAccount SubAccount;

	btcSubAccount.Currency = BTC;
	btcSubAccount.Amount, _ = strconv.ParseFloat(free["btc"].(string), 64);
	btcSubAccount.LoanAmount = 0;
	btcSubAccount.ForzenAmount, _ = strconv.ParseFloat(freezed["btc"].(string), 64);

	ltcSubAccount.Currency = LTC;
	ltcSubAccount.Amount, _ = strconv.ParseFloat(free["ltc"].(string), 64);
	ltcSubAccount.LoanAmount = 0;
	ltcSubAccount.ForzenAmount, _ = strconv.ParseFloat(freezed["ltc"].(string), 64);

	cnySubAccount.Currency = CNY;
	cnySubAccount.Amount, _ = strconv.ParseFloat(free["cny"].(string), 64);
	cnySubAccount.LoanAmount = 0;
	cnySubAccount.ForzenAmount, _ = strconv.ParseFloat(freezed["cny"].(string), 64);

	account.SubAccounts = make(map[Currency]SubAccount, 3);
	account.SubAccounts[BTC] = btcSubAccount;
	account.SubAccounts[LTC] = ltcSubAccount;
	account.SubAccounts[CNY] = cnySubAccount;

	return account, nil;
}

func (ctx * OKCoinCN_API) GetTicker(currency CurrencyPair) (*Ticker, error){
	var tickerMap map[string]interface{};
	var ticker Ticker;
	
	url := url_ticker + "?symbol=" + currencyPair2String(currency);
	bodyDataMap, err := HttpGet(url);
	if err != nil{
		return nil, err;
	}

	tickerMap = bodyDataMap["ticker"].(map[string]interface{});
	ticker.Date, _ = strconv.ParseUint(bodyDataMap["date"].(string), 10, 64);
	ticker.Last, _ = strconv.ParseFloat(tickerMap["last"].(string), 64);
	ticker.Buy, _ = strconv.ParseFloat(tickerMap["buy"].(string), 64);
	ticker.Sell, _ = strconv.ParseFloat(tickerMap["sell"].(string), 64);
	ticker.Low, _ = strconv.ParseFloat(tickerMap["low"].(string), 64);
	ticker.High, _ = strconv.ParseFloat(tickerMap["high"].(string), 64);
	ticker.Vol, _ = strconv.ParseFloat(tickerMap["vol"].(string), 64);

	return &ticker, nil;
}

func (ctx * OKCoinCN_API) GetDepth(size int, currency CurrencyPair) (*Depth, error){
	var depth Depth;
	
	url := url_depth + "?symbol=" + currencyPair2String(currency) + "&size=" + strconv.Itoa(size);
	bodyDataMap, err := HttpGet(url);
	if err != nil {
		return nil, err;
	}

	for _, v := range bodyDataMap["asks"].([]interface{}) {
		var dr DepthRecord;
		for i, vv := range v.([]interface{}) {
			switch i {
			case 0:
				dr.Price = vv.(float64);
			case 1:
				dr.Amount = vv.(float64);
			}
		}
		depth.AskList = append(depth.AskList, dr);
	}

	for _, v := range bodyDataMap["bids"].([]interface{}) {
		var dr DepthRecord;
		for i, vv := range v.([]interface{}) {
			switch i {
			case 0:
				dr.Price = vv.(float64);
			case 1:
				dr.Amount = vv.(float64);
			}
		}
		depth.BidList = append(depth.BidList, dr);
	}

	return &depth, nil;
}

func (ctx * OKCoinCN_API) GetExchangeName() string{
	return EXCHANGE_NAME;
}