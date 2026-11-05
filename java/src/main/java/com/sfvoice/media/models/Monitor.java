package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class Monitor {

    private String id;
    private String slug;
    private String text;
    @JsonProperty("project_id")  private String projectId;
    @JsonProperty("asset_class") private String assetClass;
    private Float threshold;
    private Boolean enabled;
    @JsonProperty("created_at")  private String createdAt;
    @JsonProperty("updated_at")  private String updatedAt;

    public String getId()         { return id; }
    public String getSlug()       { return slug; }
    public String getText()       { return text; }
    public String getProjectId()  { return projectId; }
    public String getAssetClass() { return assetClass; }
    public float getThreshold()   { return threshold != null ? threshold : 0.7f; }
    public boolean isEnabled()    { return enabled != null ? enabled : true; }
    public boolean getEnabled()   { return isEnabled(); }
    public String getCreatedAt()  { return createdAt; }
    public String getUpdatedAt()  { return updatedAt; }
}
