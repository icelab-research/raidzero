package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class UpdateMonitorRequest {

    private final String text;
    @JsonProperty("asset_class")
    private final String assetClass;
    private final Float threshold;
    private final Boolean enabled;

    private UpdateMonitorRequest(Builder builder) {
        this.text       = builder.text;
        this.assetClass = builder.assetClass;
        this.threshold  = builder.threshold;
        this.enabled    = builder.enabled;
    }

    public String getText()       { return text; }
    public String getAssetClass() { return assetClass; }
    public Float getThreshold()   { return threshold; }
    public Boolean getEnabled()   { return enabled; }

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private String text;
        private String assetClass;
        private Float threshold;
        private Boolean enabled;

        public Builder text(String text)             { this.text = text; return this; }
        public Builder assetClass(String assetClass) { this.assetClass = assetClass; return this; }
        public Builder threshold(float threshold)    { this.threshold = threshold; return this; }
        public Builder enabled(boolean enabled)      { this.enabled = enabled; return this; }

        public UpdateMonitorRequest build() { return new UpdateMonitorRequest(this); }
    }
}
