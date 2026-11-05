package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class MonitorEvent {

    private String id;
    @JsonProperty("monitor_id")  private String monitorId;
    @JsonProperty("document_id") private String documentId;
    @JsonProperty("asset_id")    private String assetId;
    private boolean matched;
    private Float score;
    @JsonProperty("webhook_sent") private boolean webhookSent;
    @JsonProperty("match_detail") private Object matchDetail;
    @JsonProperty("created_at")   private String createdAt;

    public String getId()          { return id; }
    public String getMonitorId()   { return monitorId; }
    public String getDocumentId()  { return documentId; }
    public String getAssetId()     { return assetId; }
    public boolean isMatched()     { return matched; }
    public Float getScore()        { return score; }
    public boolean isWebhookSent() { return webhookSent; }
    public Object getMatchDetail() { return matchDetail; }
    public String getCreatedAt()   { return createdAt; }
}
