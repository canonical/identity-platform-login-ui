import { useState, useEffect } from "react";

const CountDownText = ({
  wrapperText="",
  initialSeconds=10,
}: {
  wrapperText?: string;
  initialSeconds?: number;
}) => {
  const [seconds, setSeconds] = useState(initialSeconds);
  console.log("CountDownText rendered with seconds:", initialSeconds);
  useEffect(() => {
    console.log("CountDownText useEffect triggered with seconds:", seconds);
    const timerId = setInterval(() => {
      setSeconds((prev) => {
        const next = prev - 1;
        if (next <= 0) {
          clearInterval(timerId);
        }

        return next;
      });
      console.log("Timer tick:", seconds);
    }, 1000);
    return () => clearInterval(timerId);
  }, [initialSeconds]);

  if(seconds <= 0) {
    return null;
  }
  return `${wrapperText}${seconds.toString().padStart(2, "0")}s`
};

export default CountDownText;