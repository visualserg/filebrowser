import { now } from "./clock";

// Подсветка «недавно изменён» с учётом задержки авто-обновления.
//
// Наивный вариант «now - mtime < 10s» ломается из-за поллинга: список
// подтягивает изменения раз в ~10с, поэтому к моменту появления файла окно
// уже почти истекло и бейдж мигает пару секунд. Здесь мы отсчитываем 10с не от
// mtime, а от МОМЕНТА ПЕРВОГО ПОКАЗА файла с данным mtime - но только если на
// этот момент файл ещё «свежий» (иначе при открытии папки подсветились бы все).

// key (url|mtime) -> когда впервые увидели этот файл с этим временем изменения
const firstSeen = new Map<string, number>();

const GLOW_MS = 10000; // сколько держать подсветку от первого показа
// на момент первого показа mtime должен быть не старше этого; с запасом
// перекрывает интервал поллинга (~10с), чтобы «пойманные» опросом файлы прошли
const ELIGIBLE_MS = 12000;

function prune(t: number): void {
  for (const [k, v] of firstSeen) {
    if (t - v > GLOW_MS * 2) firstSeen.delete(k);
  }
}

// Реактивно (завязано на часы now): true, пока файл нужно подсвечивать.
export function isRecentlyModified(url: string, modified: string): boolean {
  const mtime = new Date(modified).getTime();
  if (isNaN(mtime)) return false;

  const t = now.value;
  const key = url + "|" + modified;

  let seen = firstSeen.get(key);
  if (seen === undefined) {
    const age = t - mtime;
    // на первом показе файл уже несвежий (или mtime в будущем) - не подсвечиваем
    if (age < 0 || age > ELIGIBLE_MS) return false;
    seen = t;
    firstSeen.set(key, seen);
    if (firstSeen.size > 1000) prune(t);
  }

  return t - seen < GLOW_MS;
}
