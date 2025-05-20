// src/lib/utils/timeAgo.ts

const intervals = [
  { label: "y", seconds: 31536000 },
  { label: "mo", seconds: 2592000 }, // Approx 30 days
  { label: "w", seconds: 604800 },
  { label: "d", seconds: 86400 },
  { label: "h", seconds: 3600 },
  { label: "m", seconds: 60 },
  { label: "s", seconds: 1 },
];

export function timeAgo(
  dateInput: string | Date | { seconds: number; nanos: number }
): string {
  let date: Date;

  if (typeof dateInput === "string") {
    date = new Date(dateInput);
  } else if (dateInput instanceof Date) {
    date = dateInput;
  } else if (
    dateInput &&
    typeof dateInput === "object" &&
    "seconds" in dateInput &&
    "nanos" in dateInput
  ) {
    // Handle protobuf timestamp (seconds + nanos)
    const seconds = dateInput.seconds;
    const nanos = dateInput.nanos;
    // Convert seconds to milliseconds and add nanos (converted to milliseconds)
    const milliseconds = seconds * 1000 + Math.floor(nanos / 1_000_000); // nanos to milliseconds
    date = new Date(milliseconds);
  } else {
    throw new Error("Invalid date input format");
  }

  const secondsElapsed = Math.floor((Date.now() - date.getTime()) / 1000);

  // Less than 30 seconds ago? -> "now" or actual seconds
  if (secondsElapsed < 30) {
    // return 'now';
    return `${secondsElapsed}s`; // Use seconds for very recent
  }

  const interval = intervals.find((i) => secondsElapsed >= i.seconds);

  if (interval) {
    const count = Math.floor(secondsElapsed / interval.seconds);
    return `${count}${interval.label}`;
  }

  return "just now"; // Fallback
}

export function timeAgoProfile(
  dateInput:
    | string
    | {
        seconds: number;
        nanos: number;
      }
): string {
  if (
    dateInput &&
    typeof dateInput === "object" &&
    "seconds" in dateInput &&
    "nanos" in dateInput
  ) {
    const milliseconds =
      dateInput.seconds * 1000 + Math.floor(dateInput.nanos / 1_000_000);
    const date = new Date(milliseconds);

    const options: Intl.DateTimeFormatOptions = {
      year: "numeric",
      month: "long",
    };
    return date.toLocaleDateString(undefined, options);
  }
  return "Just Now";
}
