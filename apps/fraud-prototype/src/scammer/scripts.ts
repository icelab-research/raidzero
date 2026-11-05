const scripts = {
   gift_cards_grandparent: {
      label: "gift cards grandparent",
      opening:
         "You are simulating a gift-card emergency scam. Keep it short and realistic for detector testing.",
   },
   irs: {
      label: "irs payment threat",
      opening:
         "You are simulating an IRS payment threat scam. Keep it short and realistic for detector testing.",
   },
} as const;

export type ScriptId = keyof typeof scripts;

export const STOP_TEST = "STOP TEST";

export function ids(): ScriptId[] {
   return Object.keys(scripts) as ScriptId[];
}

export function isScriptId(value: string): value is ScriptId {
   return value in scripts;
}

export function getScript(id: ScriptId) {
   return scripts[id];
}
