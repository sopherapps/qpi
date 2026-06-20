describe("Bookings — Cancel Booking", () => {
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

  it("cancels a booking after confirming the dialog", () => {
    // First, create a booking
    cy.contains("button", "Book Time Slot").click();

    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() + 30 * 60000);
    const end = new Date(Date.now() + 40 * 60000);

    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);
    cy.get("form").submit();

    // Verify the booking exists
    cy.contains("td", "user@example.com").should("be.visible");

    // Click the cancel (trash) button
    cy.get('button svg.lucide-trash-2').first().parent().click();

    // Confirm the browser dialog
    cy.on("window:confirm", () => true);

    // The booking should disappear from the table
    cy.contains("td", "user@example.com").should("not.exist");
  });
});
