package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

public class SearchResponse {

    @JsonProperty("search_query_id") private String searchQueryId;
    private List<SearchResult> results;
    @JsonProperty("page_info") private PageInfo pageInfo;

    public String getSearchQueryId()         { return searchQueryId; }
    public List<SearchResult> getResults() { return results; }
    public PageInfo getPageInfo()          { return pageInfo; }
}
