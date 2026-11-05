package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class Prefix {
    private String prefix;
    @JsonProperty("owned_by_caller") private boolean ownedByCaller;

    public String getPrefix() { return prefix; }
    public boolean isOwnedByCaller() { return ownedByCaller; }
}
