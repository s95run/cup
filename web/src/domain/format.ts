export function formatFinish(sec: number | null | undefined): string {
  if (sec == null) return "—";
  const minutes = Math.floor(sec / 60);
  const seconds = sec % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

export function formatAvgDS(ds: number | null | undefined): string {
  if (ds == null) return "—";
  const seconds = Math.floor(ds / 10);
  const tenth = ds % 10;
  const minutes = Math.floor(seconds / 60);
  return `${minutes}:${String(seconds % 60).padStart(2, "0")}.${tenth}`;
}

export function formatStageDate(iso: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(iso));
}

export function registrationCountdown(closesAt: string, now = Date.now()): string {
  const diff = new Date(closesAt).getTime() - now;
  if (Number.isNaN(diff) || diff <= 0) return "регистрация закрыта";
  const days = Math.floor(diff / 86_400_000);
  const hours = Math.floor((diff % 86_400_000) / 3_600_000);
  if (days > 0) return `до закрытия ${days} дн. ${hours} ч.`;
  const minutes = Math.floor((diff % 3_600_000) / 60_000);
  return `до закрытия ${hours} ч. ${minutes} мин.`;
}