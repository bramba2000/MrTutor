import { Container, Paper, Title, Text } from "@mantine/core";
import classes from "./TitleLayout.module.css";

interface TitleLayoutProps {
  title: string;
  subtitle?: string;
  subtitleLink?: React.ReactNode;
  children?: React.ReactNode;
}

export function TitleLayout({
  title,
  subtitle,
  children,
  subtitleLink = null,
}: TitleLayoutProps) {
  return (
    <Container size={420} my={40}>
      <Title ta="center" className={classes.title}>
        {title}
      </Title>

      <Text className={classes.subtitle}>
        {subtitle} {subtitleLink}
      </Text>

      <Paper withBorder shadow="sm" p={22} mt={30} radius="md">
        {children}
      </Paper>
    </Container>
  );
}
