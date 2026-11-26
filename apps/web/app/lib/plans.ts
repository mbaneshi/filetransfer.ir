// The three prepaid plans. This table is the single source for the price list and the legal pages.
// The API will enforce the same numbers (docs/superpowers/specs/2026-11-25-prepaid-plans-and-legal-proposal.md, section 3.1).
export type Plan = {
  id: "basic" | "max" | "ultra";
  name: string;
  storageGB: number;
  downloadGB: number;
  maxFileGB: number;
  retentionDays: number;
  priceToman: number;
};

export const PLAN_DAYS = 30;
export const GRACE_DAYS = 3;

export const PLANS: Plan[] = [
  { id: "basic", name: "پایه", storageGB: 10, downloadGB: 20, maxFileGB: 5, retentionDays: 7, priceToman: 65_000 },
  { id: "max", name: "مکس", storageGB: 50, downloadGB: 100, maxFileGB: 20, retentionDays: 14, priceToman: 315_000 },
  { id: "ultra", name: "اولترا", storageGB: 200, downloadGB: 400, maxFileGB: 50, retentionDays: 30, priceToman: 1_255_000 },
];

export const fa = (n: number) => n.toLocaleString("fa-IR");
