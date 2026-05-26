import VisibilityOffOutlinedIcon from "@mui/icons-material/VisibilityOffOutlined";
import VisibilityOutlinedIcon from "@mui/icons-material/VisibilityOutlined";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import TextField, { type TextFieldProps } from "@mui/material/TextField";
import { useState } from "react";

type SecretTextFieldProps = Omit<TextFieldProps, "type"> & {
  hideLabel?: string;
  showLabel?: string;
};

export function SecretTextField({
  hideLabel,
  showLabel,
  slotProps,
  ...props
}: SecretTextFieldProps) {
  const [isVisible, setIsVisible] = useState(false);
  const label = typeof props.label === "string" ? props.label.toLowerCase() : "secret";
  const showSecretLabel = showLabel ?? `Show ${label}`;
  const hideSecretLabel = hideLabel ?? `Hide ${label}`;

  return (
    <TextField
      {...props}
      type={isVisible ? "text" : "password"}
      slotProps={{
        ...slotProps,
        input: {
          ...slotProps?.input,
          endAdornment: (
            <InputAdornment position="end">
              <IconButton
                aria-label={isVisible ? hideSecretLabel : showSecretLabel}
                edge="end"
                onClick={() => setIsVisible((current) => !current)}
                onMouseDown={(event) => event.preventDefault()}
              >
                {isVisible ? <VisibilityOffOutlinedIcon /> : <VisibilityOutlinedIcon />}
              </IconButton>
            </InputAdornment>
          ),
        },
      }}
    />
  );
}
