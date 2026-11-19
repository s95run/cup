import { describe, expect, it } from "vitest";
import { formatAvgDS, formatFinish, registrationCountdown } from "./format";
import { isQualified, rosterSummary, splitProtocol, type SlotAthlete } from "./roster";

function athlete(partial: Partial<SlotAthlete> & Pick<SlotAthlete, "id" | "gender">): SlotAthlete {
  return {
    code: partial.id + 770_000_000,
    name: "Атлет",
    home_event_code: "x",
    home_event_name: "X",
    club_name: "",
    runs_count: 1,
    volunteering_count: 0,
    exp: 0,
    sig: "test",
    ...partial,
  };
}

describe("qualification and roster", () => {
  it("accepts a finish or four volunteer shifts", () => {
    expect(isQualified(1, 0)).toBe(true);
    expect(isQualified(0, 4)).toBe(true);
    expect(isQualified(0, 3)).toBe(false);
  });

  it("is ready only with five people, two women and no duplicates", () => {
    const slots = [
      athlete({ id: 1, gender: "female" }),
      athlete({ id: 2, gender: "female", runs_count: 0, volunteering_count: 4 }),
      athlete({ id: 3, gender: "male" }),
      athlete({ id: 4, gender: "male" }),
      athlete({ id: 5, gender: "male" }),
    ];
    expect(rosterSummary(slots).ready).toBe(true);
    expect(rosterSummary(slots.slice(0, 4)).sizeOk).toBe(false);
    expect(rosterSummary([athlete({ id: 1, gender: "female" }), ...slots.slice(2)]).femalesOk).toBe(false);
    const dup = [...slots];
    dup[4] = athlete({ id: 1, gender: "male" });
    expect(rosterSummary(dup).duplicate).toBe(true);
    expect(rosterSummary(dup).ready).toBe(false);
  });

  it("splits a protocol into classified and DSQ", () => {
    const split = splitProtocol([
      { disqualified: false, rank: 1 },
      { disqualified: true, rank: null },
    ]);
    expect(split.classified).toHaveLength(1);
    expect(split.dsq).toHaveLength(1);
    expect(split.hasResults).toBe(true);
  });
});

describe("time formatting", () => {
  it("renders finish and average", () => {
    expect(formatFinish(1520)).toBe("25:20");
    expect(formatFinish(null)).toBe("—");
    expect(formatAvgDS(14600)).toBe("24:20.0");
  });

  it("counts down to the registration deadline", () => {
    const closes = "2026-11-28T06:00:00Z";
    const now = Date.parse("2026-11-18T06:00:00Z");
    expect(registrationCountdown(closes, now)).toBe("до закрытия 10 дн. 0 ч.");
    expect(registrationCountdown(closes, Date.parse(closes))).toBe("регистрация закрыта");
  });
});