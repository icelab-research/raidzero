package icu.raidzero.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class PageInfo {

    private long total;
    private int page;
    private int limit;
    @JsonProperty("next_page_token") private String nextPageToken;

    public long getTotal()            { return total; }
    public int getPage()              { return page; }
    public int getLimit()             { return limit; }
    public String getNextPageToken()  { return nextPageToken; }
}
