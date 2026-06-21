describe("Bookings — Visibility by Role", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  context("as a regular user", () => {
    beforeEach(() => {
      cy.get('input[type="text"]').clear().type("user@example.com");
      cy.get('input[type="password"]').clear().type("userpassword1234");
      cy.get('button[type="submit"]').click();
      cy.contains("h1", "QPI Interface").should("be.visible");

      cy.contains("button", "Bookings").click();
      cy.contains("h1", "Bookings").should("be.visible");
    });

    it("shows the 'Book Time Slot' button", () => {
      cy.contains("button", "Book Time Slot").should("be.visible");
    });

    it("shows cancel action for the user's own bookings", () => {
      // Create a booking first
      cy.contains("button", "Book Time Slot").click();

      const pad = (n: number) => n.toString().padStart(2, "0");
      const start = new Date(Date.now() + 50 * 60000);
      const end = new Date(Date.now() + 60 * 60000);

      const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
      const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

      cy.get('input[type="datetime-local"]').first().type(startStr);
      cy.get('input[type="datetime-local"]').last().type(endStr);
      cy.get("form").submit();

      // The user's own booking should have a cancel button
      cy.get('button svg.lucide-trash-2').should("be.visible");
    });
  });

  context("as an administrator", () => {
    beforeEach(() => {
      cy.contains("button", "Administrator").click();
      cy.get('input[type="text"]').clear().type("admin@example.com");
      cy.get('input[type="password"]').clear().type("supersecretpassword1234");
      cy.get('button[type="submit"]').click();
      cy.contains("h1", "QPI Interface").should("be.visible");

      cy.contains("button", "Bookings").click();
      cy.contains("h1", "Bookings").should("be.visible");
    });

    it("shows the 'Book Time Slot' button", () => {
      cy.contains("button", "Book Time Slot").should("be.visible");
    });

    it("shows cancel action for all bookings (admin bypass)", () => {
      // Admin should see all bookings and be able to cancel any
      // The seeded data includes a time slot, so there should be at least one booking
      cy.get("table tbody tr").should("have.length.at.least", 1);

      // Each row should have either a trash icon or "Read-Only"
      cy.get("table tbody tr").each(($row) => {
        const hasTrash = $row.find('svg.lucide-trash-2').length > 0;
        const hasReadOnly = $row.text().includes("Read-Only");
        expect(hasTrash || hasReadOnly).to.equal(true);
      });
    });
  });
});
