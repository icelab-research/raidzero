package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SearchResult {

    @JsonProperty("document_id")   private String documentId;
    @JsonProperty("asset_id")      private String assetId;
    @JsonProperty("asset_class")   private String assetClass;
    @JsonProperty("twelvelabs_media_id") private String twelveLabsMediaId;
    private float score;
    @JsonProperty("start_ms")      private long startMs;
    @JsonProperty("end_ms")        private long endMs;
    @JsonProperty("match_type")    private String matchType;
    private String text;
    @JsonProperty("thumbnail_url") private String thumbnailUrl;

    public String getDocumentId()   { return documentId; }
    public String getAssetId()      { return assetId; }
    public String getAssetClass()   { return assetClass; }
    public String getTwelveLabsMediaId() { return twelveLabsMediaId; }
    public float getScore()         { return score; }
    public long getStartMs()        { return startMs; }
    public long getEndMs()          { return endMs; }
    public String getMatchType()    { return matchType; }
    public String getText()         { return text; }
    public String getThumbnailUrl() { return thumbnailUrl; }
}
