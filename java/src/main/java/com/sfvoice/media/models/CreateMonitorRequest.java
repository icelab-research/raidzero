package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class CreateMonitorRequest {

    private final String text;
    private final String slug;
    @JsonProperty("project_id")
    private final String projectId;
    @JsonProperty("asset_class")
    private final String assetClass;
    private final Float threshold;

    private CreateMonitorRequest(Builder builder) {
        this.text       = builder.text;
        this.slug       = builder.slug;
        this.projectId  = builder.projectId;
        this.assetClass = builder.assetClass;
        this.threshold  = builder.threshold;
    }

    public String getText()       { return text; }
    public String getSlug()       { return slug; }
    public String getProjectId()  { return projectId; }
    public String getAssetClass() { return assetClass; }
    public Float getThreshold()   { return threshold; }

    public static Builder text(String text) { return new Builder(text); }

    public static class Builder {
        private final String text;
        private String slug;
        private String projectId;
        private String assetClass;
        private Float threshold;

        private Builder(String text) { this.text = text; }

        public Builder slug(String slug)             { this.slug = slug; return this; }
        public Builder projectId(String projectId)   { this.projectId = projectId; return this; }
        public Builder assetClass(String assetClass) { this.assetClass = assetClass; return this; }
        public Builder threshold(float threshold)    { this.threshold = threshold; return this; }

        public CreateMonitorRequest build() { return new CreateMonitorRequest(this); }
    }
}
