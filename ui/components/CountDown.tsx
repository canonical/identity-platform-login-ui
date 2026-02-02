import { useState, useEffect } from "react";

const CountDown = ({
  initialSeconds,
  wrapperText,
}: {
  initialSeconds: number;
  wrapperText: string;
}) => {
  const [seconds, setSeconds] = useState(initialSeconds);

  useEffect(() => {
    // 1. If the timer reaches 0, stop the interval
    if (seconds <= 0) return;

    // 2. Set up the interval
    const timerId = setInterval(() => {
      setSeconds((prev) => prev - 1);
    }, 1000);

    // 3. Cleanup: Clear interval if the component unmounts
    // or before the effect runs again
    return () => clearInterval(timerId);
  }, [seconds]);

  return seconds > 0 ? `${wrapperText}${seconds}s` : "";
};

export default CountDown;
