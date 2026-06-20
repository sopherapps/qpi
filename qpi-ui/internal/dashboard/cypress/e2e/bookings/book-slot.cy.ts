describe("Bookings — Book a Time Slot", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Bookings").click();
    cy.contains("h1", "Bookings").should("be.visible");
  });

  it("opens the book slot modal", () => {
    cy.contains("button", "Book Time Slot").click();

    cy.contains("h3", "Book Time Slot").should("be.visible");
    cy.get('input[type="datetime-local"]').should("have.length", 2);
    cy.contains("button", "Schedule Slot").should("be.visible");
  });

  it("books a slot with valid times and shows it in the table", () => {
    cy.contains("button", "Book Time Slot").click();

    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() + 10 * 60000);
    const end = new Date(Date.now() + 20 * 60000);

    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);
    cy.get("form").submit();

    // The modal closes
    cy.contains("h3", "Book Time Slot").should("not.exist");

    // The table shows the booking with the current user's email
    cy.contains("td", "user@example.com").should("be.visible");
  });
});
