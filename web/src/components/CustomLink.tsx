import * as React from "react";
import { createLink, type LinkComponent } from "@tanstack/react-router";
import { Anchor, type AnchorProps } from "@mantine/core";

interface MantineAnchorProps extends Omit<AnchorProps, "href"> {}

const MantineLinkComponent = React.forwardRef<
  HTMLAnchorElement,
  MantineAnchorProps
>((props, ref) => {
  return <Anchor ref={ref} {...props} />;
});

const CreatedLinkComponent = createLink(MantineLinkComponent);

export type CustomLinkProps = React.ComponentProps<typeof MantineLinkComponent>;

export const CustomLink: LinkComponent<typeof MantineLinkComponent> = (
  props,
) => {
  return <CreatedLinkComponent preload="intent" {...props} />;
};
