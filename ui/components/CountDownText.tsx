import { useState, useEffect } from "react";

const CountDownText = ({
  initialSeconds,
  wrapperText,
}: {
  initialSeconds: number;
  wrapperText: string;
}) => {
  const [seconds, setSeconds] = useState(initialSeconds);
  console.log("CountDownText rendered with seconds:", initialSeconds);
  useEffect(() => {
    // 1. If the timer reaches 0, stop the interval
    if (seconds <= 0) return;

    // 2. Set up the interval
    const timerId = setInterval(() => {
      setSeconds((prev) => prev - 1);
      console.log("Timer tick:", seconds);
    }, 1000);

    // 3. Cleanup: Clear interval if the component unmounts
    // or before the effect runs again
    return () => clearInterval(timerId);
  }, [initialSeconds]);

  return seconds > 0
    ? `${wrapperText}${seconds.toString().padStart(2, "0")}s`
    : "";
};

export default CountDownText;
