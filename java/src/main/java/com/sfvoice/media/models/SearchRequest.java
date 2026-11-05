package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class SearchRequest {

    private final String query;
    private final List<String> types;
    @JsonProperty("asset_ids")
    private final List<String> assetIds;
    @JsonProperty("asset_class") private final String assetClass;
    @JsonProperty("project_id") private final String projectId;
    private final String prefix;
    private final Double threshold;
    private final Integer page;
    private final Integer limit;

    private SearchRequest(Builder builder) {
        this.query     = builder.query;
        this.types     = builder.types;
        this.assetIds  = builder.assetIds;
        this.assetClass = builder.assetClass;
        this.projectId = builder.projectId;
        this.prefix = builder.prefix;
        this.threshold = builder.threshold;
        this.page      = builder.page;
        this.limit     = builder.limit;
    }

    public String getQuery()          { return query; }
    public List<String> getTypes()    { return types; }
    public List<String> getAssetIds() { return assetIds; }
    public String getAssetClass()     { return assetClass; }
    public String getProjectId()      { return projectId; }
    public String getPrefix()         { return prefix; }
    public Double getThreshold()      { return threshold; }
    public Integer getPage()          { return page; }
    public Integer getLimit()         { return limit; }

    public static Builder query(String query) { return new Builder(query); }

    public static class Builder {
        private final String query;
        private List<String> types;
        private List<String> assetIds;
        private String assetClass;
        private String projectId;
        private String prefix;
        private Double threshold;
        private Integer page;
        private Integer limit;

        private Builder(String query) { this.query = query; }

        public Builder types(List<String> types)        { this.types = types; return this; }
        public Builder assetIds(List<String> assetIds)  { this.assetIds = assetIds; return this; }
        public Builder assetClass(String assetClass)     { this.assetClass = assetClass; return this; }
        public Builder projectId(String projectId)       { this.projectId = projectId; return this; }
        public Builder prefix(String prefix)             { this.prefix = prefix; return this; }
        public Builder threshold(double threshold)      { this.threshold = threshold; return this; }
        public Builder page(int page)                   { this.page = page; return this; }
        public Builder limit(int limit)                 { this.limit = limit; return this; }

        public SearchRequest build() { return new SearchRequest(this); }
    }
}
