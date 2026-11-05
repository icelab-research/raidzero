// pure regex/keyword scoring over a call transcript. no I/O.
// scores in 0..1 with a list of matched rule labels for evidence.
//
// the `STOP_TEST` rule is special — a hard `1.0` operator override so
// that even if the LLM scammer ignores its in-prompt safety hatch the
// responder still fires and hangs up.

import { STOP_TEST } from "../scammer/scripts.ts";

export type RuleLabel =
  | "operator_stop"
  | "gift_cards"
  | "branded_gift_card"
  | "wire_transfer"
  | "crypto"
  | "money_remit"
  | "irs"
  | "ssn"
  | "warrant_threat"
  | "authority_title"
  | "fake_tech_support"
  | "customs"
  | "fake_bank_dept"
  | "urge_secrecy"
  | "urgency"
  | "keep_on_line"
  | "remote_access"
  | "victim_buy_intent"
  | "victim_reads_card"
  | "family_emergency"
  | "bail_jail";

interface Rule {
  re: RegExp;
  weight: number;
  label: RuleLabel;
}

// keep this list sorted by category for legibility; weights don't sum,
// the aggregate is `max`.
const RULES: ReadonlyArray<Rule> = [
  // operator safety hatch — full weight, always wins.
  { re: /\bSTOP\s+TEST\b/i, weight: 1.0, label: "operator_stop" },

  // payment-method tells — strong fraud signal regardless of speaker.
  { re: /\bgift\s*cards?\b/i, weight: 0.7, label: "gift_cards" },
  {
    re: /\b(google|apple|itunes|steam|amazon|target|walmart)\s+(card|gift)/i,
    weight: 0.7,
    label: "branded_gift_card",
  },
  { re: /\bwire\s+(transfer|funds?|money)/i, weight: 0.6, label: "wire_transfer" },
  {
    re: /\b(bitcoin|btc|crypto(currency)?|usdt|tether|stable\s*coin)/i,
    weight: 0.55,
    label: "crypto",
  },
  // grouped so both alternatives are word-bounded — the prior `\bwestern\s+union|moneygram\b`
  // applied \b asymmetrically and matched `moneygram` as a suffix in larger words.
  { re: /\b(western\s+union|moneygram)\b/i, weight: 0.7, label: "money_remit" },

  // authority/agency impersonation.
  { re: /\b(irs|tax\s+(office|authority|department))\b/i, weight: 0.6, label: "irs" },
  { re: /\bsocial\s+security|ssn\b/i, weight: 0.55, label: "ssn" },
  {
    re: /\b(warrant|criminal\s+(charges|complaint))\b/i,
    weight: 0.7,
    label: "warrant_threat",
  },
  { re: /\b(officer|agent|inspector|detective)\s+[a-z]{3,}\b/i, weight: 0.25, label: "authority_title" },
  {
    re: /\b(microsoft|apple|google|amazon)\s+(security|support|technician)/i,
    weight: 0.55,
    label: "fake_tech_support",
  },
  { re: /\bcustoms\s+(officer|department|hold|duty)/i, weight: 0.55, label: "customs" },
  { re: /\b(your\s+)?(bank|fraud)\s+(department|team)\b/i, weight: 0.4, label: "fake_bank_dept" },

  // urgency / secrecy / social-engineering markers.
  { re: /\bdo(n'?t|\s+not)\s+(tell|hang\s+up|talk\s+to)/i, weight: 0.5, label: "urge_secrecy" },
  {
    re: /\b(immediately|right\s+now|within\s+the\s+next\s+(hour|few\s+minutes|minutes))\b/i,
    weight: 0.35,
    label: "urgency",
  },
  { re: /\bstay\s+on\s+the\s+(line|phone)\b/i, weight: 0.45, label: "keep_on_line" },
  {
    re: /\bremote\s+(desktop|access|control)|anydesk|teamviewer|ammyy/i,
    weight: 0.7,
    label: "remote_access",
  },

  // victim-side compliance markers.
  { re: /\bi'?ll\s+go\s+buy\b/i, weight: 0.55, label: "victim_buy_intent" },
  { re: /\bhere'?s\s+the\s+(card\s+)?(number|code)/i, weight: 0.7, label: "victim_reads_card" },

  // grandparent / family-emergency tells.
  {
    re: /\b(your\s+)?(grandson|granddaughter|nephew|niece)\b/i,
    weight: 0.4,
    label: "family_emergency",
  },
  { re: /\b(bail|jail|posted\s+bond)\b/i, weight: 0.45, label: "bail_jail" },
];

export interface ScoreResult {
  score: number;
  labels: RuleLabel[];
}

export type Turn = {
  role: "user" | "assistant" | string;
  text: string;
};

/** score a string or full transcript (list of turns). returns the max
 *  matching weight + the list of triggered rule labels. */
export function score(input: string | ReadonlyArray<Turn>): ScoreResult {
  const text = typeof input === "string" ? input : joinTranscript(input);
  let max = 0;
  const labels: RuleLabel[] = [];
  for (const rule of RULES) {
    if (rule.re.test(text)) {
      labels.push(rule.label);
      if (rule.weight > max) max = rule.weight;
    }
  }
  return { score: max, labels };
}

/** true if the latest turn alone contains the operator stop phrase. */
export function isStopTest(text: unknown): boolean {
  return typeof text === "string" && text.toUpperCase().includes(STOP_TEST);
}

function joinTranscript(turns: ReadonlyArray<Turn>): string {
  return turns.map((t) => `${t.role}: ${t.text}`).join("\n");
}

export function rules(): ReadonlyArray<Rule> {
  return RULES;
}
