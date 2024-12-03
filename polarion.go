package polarion_wsdl

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/AVaitkunas/polarion-wsdl/session_ws"
	"github.com/AVaitkunas/polarion-wsdl/test_ws"
	"github.com/AVaitkunas/polarion-wsdl/tracker_ws"

	"github.com/hooklift/gowsdl/soap"
)

// soap envelope header containing session ID
// should be included in all requests to API (handled in Polarion constuctor)
type sessionHeader struct {
	XMLName        xml.Name `xml:"http://ws.polarion.com/session sessionID"`
	Value          string   `xml:",chardata"`
	Actor          string   `xml:"http://schemas.xmlsoap.org/soap/envelope/ actor,attr"`
	MustUnderstand string   `xml:"http://schemas.xmlsoap.org/soap/envelope/ mustUnderstand,attr"`
}

func newSessionHeader(sessionID string) *sessionHeader {
	return &sessionHeader{
		Value:          sessionID,
		Actor:          "http://schemas.xmlsoap.org/soap/actor/next",
		MustUnderstand: "0",
	}
}

type Polarion struct {

	// http client which is shared across all soap clients
	HttpClient    *http.Client
	SessionClient *soap.Client
	SessionWS     session_ws.SessionWebService
	TrackerClient *soap.Client
	TrackerWS     tracker_ws.TrackerWebService
	TestClient    *soap.Client
	TestWS        test_ws.TestManagementWebService
}

func NewPolarion(polarion_url, username, accessToken string, timeout time.Duration) (*Polarion, error) {
	sessionEndpoint := fmt.Sprintf("%s/%s", polarion_url, "polarion/ws/services/SessionWebService?wsdl")
	trackerEndpoint := fmt.Sprintf("%s/%s", polarion_url, "polarion/ws/services/TrackerWebService?wsdl")
	testsEndpoint := fmt.Sprintf("%s/%s", polarion_url, "polarion/ws/services/TestManagementWebService?wsdl")

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	sessionID, err := loginWithTokenRaw(httpClient, sessionEndpoint, username, accessToken)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to login and create new session for %v: %v",
			username, err,
		)
	}

	sessionHeader := newSessionHeader(sessionID)

	sessionClient := soap.NewClient(
		sessionEndpoint,
		soap.WithHTTPClient(httpClient),
		soap.WithTimeout(timeout),
	)
	sessionClient.AddHeader(sessionHeader)
	sessionWS := session_ws.NewSessionWebService(sessionClient)

	trackerClient := soap.NewClient(
		trackerEndpoint,
		soap.WithHTTPClient(httpClient),
		soap.WithTimeout(timeout),
	)
	trackerClient.AddHeader(sessionHeader)
	trackerWS := tracker_ws.NewTrackerWebService(trackerClient)

	testClient := soap.NewClient(
		testsEndpoint,
		soap.WithHTTPClient(httpClient),
		soap.WithTimeout(timeout),
	)
	testClient.AddHeader(sessionHeader)
	testWS := test_ws.NewTestManagementWebService(testClient)

	polarion := &Polarion{
		HttpClient:    httpClient,
		SessionClient: sessionClient,
		SessionWS:     sessionWS,
		TrackerClient: trackerClient,
		TrackerWS:     trackerWS,
		TestClient:    testClient,
		TestWS:        testWS,
	}

	return polarion, nil
}

func (p *Polarion) IsLoggedIn() (bool, error) {
	req := session_ws.HasSubject{}
	resp, err := p.SessionWS.HasSubject(&req)
	if err != nil {
		return false, fmt.Errorf("failed to check login status in Poalrion: %v", err)
	}

	if resp == nil {
		return false, fmt.Errorf("unexpected nil response while checking login status in Polarion")
	}

	return resp.HasSubjectReturn, nil
}

func (p *Polarion) GetWorkItemById(
	projectId, itemId string,
) (*tracker_ws.WorkItem, error) {
	req := tracker_ws.GetWorkItemById{
		ProjectId:  projectId,
		WorkitemId: itemId,
	}

	resp, err := p.TrackerWS.GetWorkItemById(&req)
	if err != nil {
		return nil, fmt.Errorf("error getting work item %v", err)
	}

	return resp.GetWorkItemByIdReturn, nil
}

func (p *Polarion) QueryWorkItems(
	query, sortField string,
	fields []string,
) ([]*tracker_ws.WorkItem, error) {
	req := tracker_ws.QueryWorkItems{
		Query: query,
	}

	if len(fields) > 0 {
		req.Fields = fields
		req.Sort = sortField
		if sortField == "" {
			return nil, fmt.Errorf(
				"sortField should be specified if fields parameter is provided",
			)
		}
	}

	resp, err := p.TrackerWS.QueryWorkItems(&req)
	if err != nil {
		return nil, fmt.Errorf("error querying work items: %v", err)
	}

	return resp.QueryWorkItemsReturn, nil
}

func (p *Polarion) QueryWorkItemsBySQL(
	sqlQuery string,
	fields []string,
) ([]*tracker_ws.WorkItem, error) {
	req := tracker_ws.QueryWorkItemsBySQL{
		SqlQuery: sqlQuery,
		Fields:   fields,
	}

	resp, err := p.TrackerWS.QueryWorkItemsBySQL(&req)
	if err != nil {
		return nil, fmt.Errorf("error querying work items by SQL: %v", err)
	}

	return resp.QueryWorkItemsBySQLReturn, nil
}

func (p *Polarion) GetWorkItemsCount(query string) (int, error) {
	req := tracker_ws.GetWorkItemsCount{
		Query: query,
	}

	resp, err := p.TrackerWS.GetWorkItemsCount(&req)
	if err != nil {
		return -1, fmt.Errorf("error querying work items: %v", err)
	}
	return int(resp.GetWorkItemsCountReturn), nil
}

func (p *Polarion) QueryBaselines(
	query string,
	sortField string,
) ([]*tracker_ws.Baseline, error) {
	req := tracker_ws.QueryBaselines{
		Query: query,
		Sort:  sortField,
	}

	resp, err := p.TrackerWS.QueryBaselines(&req)
	if err != nil {
		return nil, fmt.Errorf("error querying baselines: %v", err)
	}

	return resp.QueryBaselinesReturn, nil
}

func (p *Polarion) GetTestCaseRecords(
	testRunUri, testCaseUri *test_ws.SubterraURI,
) ([]*test_ws.TestRecord, error) {
	req := test_ws.GetTestCaseRecords{
		TestRunUri:  testRunUri,
		TestCaseUri: testCaseUri,
	}
	resp, err := p.TestWS.GetTestCaseRecords(&req)
	if err != nil {
		return nil, err
	}
	return resp.GetTestCaseRecordsReturn, nil
}

// query syntax requires to specify project ID,
// so it's possible to get only test records for single test run in one operation
// https://docs.sw.siemens.com/en-US/doc/230235217/PL20221020258116340.xid1465510/xid1570678
func (p *Polarion) QueryTestRecords(
	query, sortField string,
	limit int,
) ([]*test_ws.TestRecord, error) {
	req := test_ws.SearchTestRecords{
		Query: query,
		Sort:  sortField,
	}

	if limit > 0 {
		req.Limit = int32(limit)
	}

	resp, err := p.TestWS.SearchTestRecords(&req)
	if err != nil {
		return nil, fmt.Errorf("test records search failed: %v", err)
	}
	return resp.SearchTestRecordsReturn, nil
}

func (p *Polarion) GetTestRunById(projectID, testRunID string) (*test_ws.TestRun, error) {
	req := test_ws.GetTestRunById{
		Project: projectID,
		Id:      testRunID,
	}
	resp, err := p.TestWS.GetTestRunById(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to get test run by id: %v", err)
	}

	return resp.GetTestRunByIdReturn, nil
}

func (p *Polarion) QueryTestRuns(
	query, sortField string,
	fields []string,
) ([]*test_ws.TestRun, error) {
	req := test_ws.SearchTestRunsWithFields{
		Query:  query,
		Sort:   sortField,
		Fields: fields,
	}
	resp, err := p.TestWS.SearchTestRunsWithFields(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to search for test runs with fields (%v): %v", fields, err)
	}
	return resp.SearchTestRunsWithFieldsReturn, nil
}

func (p *Polarion) QueryWorkItemsInBaseline(
	baselineRevision, query, sort string,
	fields []string,
) ([]*tracker_ws.WorkItem, error) {
	req := tracker_ws.QueryWorkItemsInBaseline{
		Query:            query,
		BaselineRevision: baselineRevision,
		Fields:           fields,
		Sort:             sort,
	}

	resp, err := p.TrackerWS.QueryWorkItemsInBaseline(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to query work items in baseline: %v", err)
	}

	return resp.QueryWorkItemsInBaselineReturn, nil
}

func (p *Polarion) QueryRevisions(query string, fields []string, sort string) ([]*tracker_ws.Revision, error) {
	req := tracker_ws.QueryRevisions{
		Query:  query,
		Fields: fields,
		Sort:   sort,
	}

	resp, err := p.TrackerWS.QueryRevisions(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to query revisions: %v", err)
	}

	return resp.QueryRevisionsReturn, nil
}

func (p *Polarion) QueryWorkItemsInBaselineBySQL(
	baselineRevision, sqlQuery string,
	fields []string,
) ([]*tracker_ws.WorkItem, error) {
	sqlReq := tracker_ws.QueryWorkItemsInBaselineBySQL{
		SqlQuery:         sqlQuery,
		BaselineRevision: baselineRevision,
		Fields:           []string{"id", "title", "status", "updated"},
	}

	resp, err := p.TrackerWS.QueryWorkItemsInBaselineBySQL(&sqlReq)
	if err != nil {
		return nil, fmt.Errorf("failed to query work items by SQL: %v", err)
	}

	return resp.QueryWorkItemsInBaselineBySQLReturn, nil
}

func (p *Polarion) GetCustomField(wiURI *tracker_ws.SubterraURI, key string) (*tracker_ws.CustomField, error) {
	req := tracker_ws.GetCustomField{
		WorkitemURI: wiURI,
		Key:         key,
	}

	resp, err := p.TrackerWS.GetCustomField(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to get WorkItem CustomField with key '%s': %v", key, err)
	}
	return resp.GetCustomFieldReturn, nil
}
