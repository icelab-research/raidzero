import { describe, expect, test } from "vitest";
import * as H from "../src/fraud/heuristics.ts";

describe("score — string input", () => {
  test("benign text returns 0 and no labels", () => {
    const r = H.score("Hi, I'd like to make a reservation for two on Friday.");
    expect(r.score).toBe(0);
    expect(r.labels).toEqual([]);
  });

  test("STOP TEST forces 1.0 with operator_stop label", () => {
    const r = H.score("Wait — STOP TEST.");
    expect(r.score).toBe(1);
    expect(r.labels).toContain("operator_stop");
  });

  test("STOP TEST is case-insensitive and whitespace-tolerant", () => {
    expect(H.score("stop  test now").labels).toContain("operator_stop");
    expect(H.score("Stop Test").labels).toContain("operator_stop");
  });

  test("gift card mention triggers gift_cards", () => {
    const r = H.score("we accept gift cards from Target");
    expect(r.score).toBeGreaterThanOrEqual(0.7);
    expect(r.labels).toContain("gift_cards");
  });

  test("branded gift card phrase triggers branded_gift_card", () => {
    const r = H.score("Please go to Walmart and buy an Apple gift card.");
    expect(r.score).toBeGreaterThanOrEqual(0.7);
    expect(r.labels).toContain("branded_gift_card");
  });

  test("wire transfer triggers wire_transfer", () => {
    const r = H.score("we need you to wire transfer the funds");
    expect(r.score).toBeGreaterThanOrEqual(0.6);
    expect(r.labels).toContain("wire_transfer");
  });

  test("IRS impersonation triggers irs", () => {
    const r = H.score("This is Agent Daniels with the IRS.");
    expect(r.score).toBeGreaterThanOrEqual(0.6);
    expect(r.labels).toContain("irs");
  });

  test("SSN mention triggers ssn", () => {
    expect(H.score("Please confirm your social security number.").labels).toContain("ssn");
  });

  test("warrant threat triggers warrant_threat", () => {
    const r = H.score("We have an arrest warrant for your arrest pending.");
    expect(r.score).toBeGreaterThanOrEqual(0.7);
    expect(r.labels).toContain("warrant_threat");
  });

  test("remote-access tool name triggers remote_access", () => {
    const r = H.score("Please install AnyDesk so I can clean the virus.");
    expect(r.score).toBeGreaterThanOrEqual(0.7);
    expect(r.labels).toContain("remote_access");
  });

  test("grandparent + bail combo triggers both labels", () => {
    const r = H.score("It's your grandson Michael, I need bail money.");
    expect(r.labels).toContain("family_emergency");
    expect(r.labels).toContain("bail_jail");
  });

  test("secrecy phrase triggers urge_secrecy", () => {
    expect(H.score("Please don't tell mom and dad about this call.").labels).toContain(
      "urge_secrecy",
    );
  });

  test("score returns the MAX weight, not the sum", () => {
    // gift_cards (0.7) and urgency (0.35) both match — should be 0.7, not 1.05.
    const r = H.score("Buy gift cards immediately and read me the numbers.");
    expect(r.score).toBe(0.7);
    expect(r.labels).toContain("gift_cards");
    expect(r.labels).toContain("urgency");
  });
});

describe("score — transcript list input", () => {
  test("joins roles and matches across turns", () => {
    const r = H.score([
      { role: "assistant", text: "Hello, this is Agent Daniels with the IRS." },
      { role: "user", text: "What is this about?" },
      { role: "assistant", text: "You need to buy gift cards to settle the warrant." },
    ]);
    expect(r.score).toBeGreaterThanOrEqual(0.7);
    expect(r.labels).toContain("gift_cards");
    expect(r.labels).toContain("irs");
    expect(r.labels).toContain("warrant_threat");
  });

  test("empty transcript scores 0", () => {
    const r = H.score([]);
    expect(r.score).toBe(0);
    expect(r.labels).toEqual([]);
  });
});

describe("isStopTest", () => {
  test("true for an exact STOP TEST phrase", () => {
    expect(H.isStopTest("STOP TEST")).toBe(true);
  });

  test("true case-insensitive", () => {
    expect(H.isStopTest("stop test please")).toBe(true);
  });

  test("false for unrelated text", () => {
    expect(H.isStopTest("I'd like a reservation, please.")).toBe(false);
  });

  test("false for nil / non-strings", () => {
    expect(H.isStopTest(null)).toBe(false);
    expect(H.isStopTest(123)).toBe(false);
    expect(H.isStopTest(undefined)).toBe(false);
  });
});

describe("rules", () => {
  test("exposes the rule list", () => {
    const rs = H.rules();
    expect(rs.length).toBeGreaterThan(0);
    expect(rs.some((r) => r.label === "operator_stop")).toBe(true);
  });
});
