import React, { useState, useEffect } from "react";
import { Notification } from "@canonical/react-components";

const CountDownText = ({
  wrapperText = "",
  initialSeconds = 10,
}: {
  wrapperText?: string;
  initialSeconds?: number;
}) => {
  const [seconds, setSeconds] = useState(initialSeconds);
  useEffect(() => {
    const timerId = setInterval(() => {
      setSeconds((prev) => {
        const next = prev - 1;
        if (next <= 0) {
          clearInterval(timerId);
        }
        return next;
      });
    }, 1000);
    return () => clearInterval(timerId);
  }, [initialSeconds]);

  if (seconds <= 0) {
    return null;
  }
  return (
    <Notification
      inline
      severity="positive"
      borderless
      className="u-no-margin--bottom"
    >{`${wrapperText}${seconds.toString().padStart(2, "0")}s`}</Notification>
  );
};

export default CountDownText;
